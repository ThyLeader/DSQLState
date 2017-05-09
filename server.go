package dsqlstate

import (
	"database/sql"
	"fmt"
	"github.com/vattle/sqlboiler/boil"
	"github.com/vattle/sqlboiler/queries/qm"
	"sync"
	// "encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/jonas747/discordgo"
	"github.com/jonas747/dsqlstate/models"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"gopkg.in/nullbio/null.v6"
	"reflect"
	"strconv"
	"time"
)

//go:generate sqlboiler --no-hooks -w "discord_users,discord_guilds,discord_guild_roles,discord_guild_channels,discord_private_channels,discord_members,discord_channel_overwrites,discord_voice_states,discord_messages,discord_message_revisions,discord_message_embeds" postgres

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

type memCache struct {
	SelfUser  *discordgo.User
	SessionID string
	sync.Mutex
}

// Server keeps the database up to date
type Server struct {
	self             *discordgo.User
	db               *sql.DB
	Debug            bool
	OnError          func(err error)
	OnLog            func(msg string)
	LoadAllMembers   bool
	UpdateGameStatus bool

	// Queue all events until ready
	readyShards []bool
	readyGuilds map[int64]bool
	readyLock   sync.RWMutex

	cache        memCache
	shardWorkers []*shardWorker
}

type QueuedEvent struct {
	Evt     interface{}
	Session *discordgo.Session
}

// New returns a default state using the database
func NewServer(db *sql.DB, numShards int) *Server {
	if numShards < 1 {
		numShards = 1
	}

	return &Server{
		db:             db,
		LoadAllMembers: true,
		readyShards:    make([]bool, numShards),
		readyGuilds:    make(map[int64]bool),
	}
}

// RunWorkers starts the shard workers, this is required if you want all members loaded into the db
func (s *Server) RunWorkers(numShards int) {
	if numShards < 1 {
		numShards = 1
	}

	s.shardWorkers = make([]*shardWorker, numShards)
	for i := 0; i < numShards; i++ {
		s.shardWorkers[i] = &shardWorker{
			GCCHan:   make(chan *GuildCreateEvt),
			StopChan: make(chan bool),
			server:   s,
		}
		go s.shardWorkers[i].queueHandler()
	}
}

// StopWorkers stops the member fetcher workers
func (s *Server) StopWorkers() {
	for _, v := range s.shardWorkers {
		close(v.StopChan)
	}

	s.shardWorkers = nil
}

type GuildCreateEvt struct {
	Session *discordgo.Session
	G       int64
}

type shardWorker struct {
	server   *Server
	StopChan chan bool
	GCCHan   chan *GuildCreateEvt
}

// Purpose of this queue is to send guild members requests to the gatway
// but if we do it too fast, we will get disconnected
func (s *shardWorker) queueHandler() {
	guildsToBeProcessed := make([]*GuildCreateEvt, 0)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.StopChan:
			return
		case g := <-s.GCCHan:
			guildsToBeProcessed = append(guildsToBeProcessed, g)
		case <-ticker.C:
			if len(guildsToBeProcessed) < 1 || !s.server.AllGuildsReady() {
				continue
			}
			g := guildsToBeProcessed[0]
			guildsToBeProcessed = guildsToBeProcessed[1:]

			logrus.Info("Requesting members from ", g.G)

			err := g.Session.RequestGuildMembers(strconv.FormatInt(g.G, 10), "", 0)

			if s.server.handleError(err, "Worker failed requesting guild members, retrying...") {
				guildsToBeProcessed = append(guildsToBeProcessed, g)
			}
		}
	}
}

func (s *Server) handleError(err error, message string) bool {
	if err == nil {
		return false
	}

	if s.OnError != nil {
		s.OnError(errors.Wrap(err, message))
	} else {
		logrus.WithError(err).Error(message)
	}

	return true
}

func (s *Server) AllGuildsReady() bool {
	s.readyLock.RLock()
	defer s.readyLock.RUnlock()
	for _, v := range s.readyShards {
		if !v {
			return false
		}
	}

	// logrus.Println(len(s.readyGuilds))
	if len(s.readyGuilds) > 0 {
		return false
	}

	return true
}

func (s *Server) NumNotReady() (bool, int) {
	s.readyLock.RLock()
	defer s.readyLock.RUnlock()
	b := true
	for _, v := range s.readyShards {
		if !v {
			b = false
			break
		}
	}

	n := len(s.readyGuilds)
	return b, n
}

func (s *Server) ShardsReady() bool {
	s.readyLock.RLock()
	defer s.readyLock.RUnlock()
	for _, v := range s.readyShards {
		if !v {
			return false
		}
	}

	return true
}

func (srv *Server) HandleEvent(s *discordgo.Session, evt interface{}) {

	switch t := evt.(type) {
	case *discordgo.Ready:
		srv.ready(s, t)
		return
	}

	for !srv.ShardsReady() {
		time.Sleep(time.Second)
	}

	switch t := evt.(type) {
	// Guilds
	case *discordgo.GuildCreate:
		srv.guildCreate(s, t.Guild)
		return
	case *discordgo.GuildDelete:
		srv.guildRemove(t.Guild)
		return
	}

	for !srv.AllGuildsReady() {
		// Wait until all the guilds have loaded their initial set receivied in ready and guild creates
		// In the future, check each guild individually
		// And store a queue on disk perhaps, for when discord has downtimes and makes guilds unavailable for extended perios
		// Then again we wouldnt receive many events for those guilds so we should be able to keep the short queue in memory
		// HMMMM
		time.Sleep(time.Second)
	}

	if srv.Debug {
		t := reflect.Indirect(reflect.ValueOf(evt)).Type()
		logrus.Debug("Inc event ", t.Name())
	}

	switch t := evt.(type) {
	case *discordgo.GuildUpdate:
		srv.guildUpdate(s, t.Guild)

	// Members
	case *discordgo.GuildMemberAdd:
		srv.updateMember(s, t.Member, true)
	case *discordgo.GuildMemberUpdate:
		srv.updateMember(s, t.Member, true)
	case *discordgo.GuildMemberRemove:
		models.DiscordMembers(srv.db, qm.Where("user_id = ?", t.User.ID), qm.Where("guild_id = ?", t.GuildID)).UpdateAll(models.M{"left_at": time.Now()})
	// Roles
	case *discordgo.GuildRoleCreate:
		srv.updateRole(t.GuildID, t.Role)
	case *discordgo.GuildRoleUpdate:
		srv.updateRole(t.GuildID, t.Role)
	case *discordgo.GuildRoleDelete:
		srv.removeRole(t.RoleID)

	// Channels
	case *discordgo.ChannelCreate:
		if t.Channel.GuildID != "" {
			srv.updateGuildChannel(t.Channel)
		} else if t.Channel.Recipient != nil {
			srv.updatePrivateChannel(t.Channel)
		}
	case *discordgo.ChannelUpdate:
		if t.Channel.GuildID != "" {
			srv.updateGuildChannel(t.Channel)
		} else if t.Channel.Recipient != nil {
			srv.updatePrivateChannel(t.Channel)
		}
	case *discordgo.ChannelDelete:
		if t.Channel.GuildID != "" {
			models.DiscordGuildChannels(srv.db, qm.Where("id = ?", t.Channel.ID)).UpdateAll(models.M{"deleted_at": time.Now()})
		}
	// Messages
	case *discordgo.MessageCreate:
		srv.messageCreate(t.Message)
	case *discordgo.MessageUpdate:
		srv.messageUpdate(nil, nil, t.Message, true)
	case *discordgo.MessageDelete:
		srv.messageDelete(t.Message)

	// Other
	case *discordgo.VoiceStateUpdate:
		srv.updateVoiecState(t.VoiceState)
	case *discordgo.UserUpdate:
		srv.updateUser(nil, t.User)
	case *discordgo.PresenceUpdate:
		srv.presenceUpdate(t)
	case *discordgo.GuildMembersChunk:
		srv.guildMembersChunk(t)
	}

}

func shardClause(guildColumn string, numShards, current int) string {
	// (guild_id >> 22) % num_shards == shard_id
	q := fmt.Sprintf("(%s >> 22) %% %d = %d", guildColumn, numShards, current)
	return q
}

func shardClauseAnd(guildColumn string, numShards, current int) string {
	if numShards < 2 {
		return ""
	}

	return " AnD " + shardClause(guildColumn, numShards, current)
}

func (srv *Server) ready(s *discordgo.Session, r *discordgo.Ready) {

	srv.cache.Lock()
	srv.cache.SelfUser = r.User
	srv.cache.Unlock()

	// var now = time.Now()

	// Mark all guilds on this shard as deleted
	_, err := srv.db.Exec("UPDATE discord_guilds SET left_at = $1 WHERE left_at IS NULL"+shardClauseAnd("id", s.ShardCount, s.ShardID), time.Now())
	srv.handleError(err, "Failed marking shard guilds as left")

	sc := shardClauseAnd("guild_id", s.ShardCount, s.ShardID)

	// Mark all guild roles as deleted
	_, err = srv.db.Exec("UPDATE discord_guild_roles SET deleted_at = $1 WHERE deleted_at IS NULL"+sc, time.Now())
	srv.handleError(err, "Failed marking shard guild roles as deleted")

	// Mark all guild channels as deleted
	_, err = srv.db.Exec("UPDATE discord_guild_channels SET deleted_at = $1 WHERE deleted_at IS NULL"+sc, time.Now())
	srv.handleError(err, "Failed marking shard guild channels as deleted")

	// Clear the voice srvates, as we get a new fresh set in the guild creates
	_, err = srv.db.Exec("DELETE FROM discord_voice_states" + sc)
	srv.handleError(err, "Failed marking shard guild voice_states as deleted")

	// Clear members, as people can have left in the meantime, it is now unclear who is srvill on the server
	_, err = srv.db.Exec("UPDATE discord_members SET left_at = $1 WHERE left_at IS NULL"+sc, time.Now())
	srv.handleError(err, "Failed marking shard guild members as left")

	var wg sync.WaitGroup
	for _, v := range r.Guilds {
		parsedGID, err := strconv.ParseInt(v.ID, 10, 64)
		if srv.handleError(err, "Failed parsing guild id") {
			return
		}

		srv.readyLock.Lock()
		logrus.Println(parsedGID)
		if v.Unavailable {
			srv.readyGuilds[parsedGID] = false
			logrus.Println(parsedGID, len(srv.readyGuilds))
		}
		srv.readyLock.Unlock()

		if v.Unavailable {
			srv.db.Exec("UPDATE discord_guilds SET left_at = NULL WHERE id = $1", v.ID)
		} else {
			logrus.Info(parsedGID, "not unavail")
			wg.Add(1)
			go func(g *discordgo.Guild) {
				srv.guildCreate(s, g)
				wg.Done()
			}(v)
		}
	}

	wg.Wait()

	srv.readyLock.Lock()
	srv.readyShards[s.ShardID] = true
	srv.readyLock.Unlock()
}

func (srv *Server) guildMembersChunk(chunk *discordgo.GuildMembersChunk) {
	started := time.Now()
	for _, v := range chunk.Members {
		v.GuildID = chunk.GuildID
		srv.updateMember(nil, v, true)
	}
	logrus.Debug("Updated ", len(chunk.Members), " in ", time.Since(started))
}

func (srv *Server) guildCreate(session *discordgo.Session, g *discordgo.Guild) {
	parsedGID, _ := strconv.ParseInt(g.ID, 10, 64)

	defer func() {
		srv.readyLock.Lock()
		delete(srv.readyGuilds, parsedGID)
		srv.readyLock.Unlock()
	}()

	if srv.LoadAllMembers && g.Large {
		go func(gid int64) {
			worker := 0
			if session.ShardCount > 0 {
				worker = int((gid >> 22) % int64(session.ShardCount))
			}
			srv.shardWorkers[worker].GCCHan <- &GuildCreateEvt{
				G:       gid,
				Session: session,
			}
		}(parsedGID)
	}

	logrus.Debug("GC! ", g.Name)
	srv.guildUpdate(session, g)

	// Update all the members and users
	for _, v := range g.Members {
		srv.updateMember(session, v, true)
	}
}

func (srv *Server) guildRemove(g *discordgo.Guild) {
	parsedID, _ := strconv.ParseInt(g.ID, 10, 64)
	srv.readyLock.Lock()
	delete(srv.readyGuilds, parsedID)
	srv.readyLock.Unlock()
	srv.db.Exec("UPDATE discord_guilds SET left_at = $1 WHERE id = $2", time.Now(), g.ID)
}

func (srv *Server) guildUpdate(session *discordgo.Session, g *discordgo.Guild) {
	parsedId, err := strconv.ParseInt(g.ID, 10, 64)
	if err != nil {
		logrus.Printf("%#v\n", g)
		panicErr(err)
	}

	// Servers cna have no owner in edge cases...
	ownerID, _ := strconv.ParseInt(g.OwnerID, 10, 64)

	var parsedAFK int64
	var embedChannel int64

	if g.AfkChannelID != "" {
		parsedAFK, err = strconv.ParseInt(g.AfkChannelID, 10, 64)
		if err != nil {
			panic(err)
		}
	}

	if g.EmbedChannelID != "" {
		embedChannel, err = strconv.ParseInt(g.EmbedChannelID, 10, 64)
		if err != nil {
			panic(err)
		}
	}

	model := &models.DiscordGuild{
		ID: parsedId,

		OwnerID: ownerID,

		Name:   g.Name,
		Icon:   g.Icon,
		Region: g.Region,

		EmbedEnabled:   g.EmbedEnabled,
		EmbedChannelID: embedChannel,
		AfkChannelID:   parsedAFK,
		AfkTimeout:     g.AfkTimeout,

		Splash:            g.Splash,
		MemberCount:       g.MemberCount,
		VerificationLevel: int16(g.VerificationLevel),
		Large:             g.Large,
		DefaultMessageNotifications: int16(g.DefaultMessageNotifications),
	}

	err = model.Upsert(srv.db, true, []string{"id"}, []string{"name", "icon", "region", "afk_channel_id", "embed_channel_id", "owner_id", "splash", "afk_timeout", "member_count", "verification_level", "embed_enabled", "large", "default_message_notifications", "left_at"})
	srv.handleError(err, "Failed upserting guild")

	// Update all roles
	for _, v := range g.Roles {
		srv.updateRole(g.ID, v)
	}

	// Update all channels
	for _, v := range g.Channels {
		srv.updateGuildChannel(v)
	}
}

func (s *Server) updateUser(tx *sql.Tx, user *discordgo.User) error {

	exec := boil.Executor(tx)
	if exec == nil {
		exec = s.db
	}

	parsedId, err := strconv.ParseInt(user.ID, 10, 64)
	panicErr(err)
	model := &models.DiscordUser{
		ID:            parsedId,
		Username:      user.Username,
		Discriminator: user.Discriminator,
		Bot:           user.Bot,
		Avatar:        user.Avatar,
	}

	err = model.Upsert(exec, true, []string{"id"}, []string{"username", "discriminator", "bot", "avatar"})
	return err
}

func (s *Server) presenceUpdate(p *discordgo.PresenceUpdate) {
	// if p.User.Username == "" && !s.UpdateGameStatus {
	// 	return
	// }

	parsedId, err := strconv.ParseInt(p.User.ID, 10, 64)
	panicErr(err)

	model := &models.DiscordUser{
		ID:     parsedId,
		Status: string(p.Status),
	}

	columns := []string{}
	if p.User.Username != "" {
		model.Username = p.User.Username
		model.Discriminator = p.User.Discriminator
		model.Avatar = p.User.Avatar
		model.Bot = p.User.Bot

		columns = append(columns, "username", "discriminator", "bot", "avatar")
	}

	if p.Status != "" {
		columns = append(columns, "status")
	}

	if p.Game != nil {
		model.GameName = null.StringFrom(p.Game.Name)
		model.GameType = null.IntFrom(p.Game.Type)
		model.GameURL = null.StringFrom(p.Game.URL)

		columns = append(columns, "game_name", "game_type", "game_url")
	}
	err = model.Upsert(s.db, true, []string{"id"}, columns)
	s.handleError(err, "Failed upserting presence")
}

func (s *Server) updateMember(session *discordgo.Session, member *discordgo.Member, updtUser bool) {
	// Update roles
	transaction, err := s.db.Begin()
	if s.handleError(err, "Failed starting a transaction to updating member") {
		return
	}

	if updtUser {
		err = s.updateUser(transaction, member.User)
		if s.handleError(err, "Failed updating user") {
			transaction.Rollback()
			return
		}
	}

	parsedMID, err := strconv.ParseInt(member.User.ID, 10, 64)
	panicErr(err)
	parsedGID, err := strconv.ParseInt(member.GuildID, 10, 64)
	panicErr(err)

	joinedParsed, _ := discordgo.Timestamp(member.JoinedAt).Parse()
	model := &models.DiscordMember{
		UserID:  parsedMID,
		GuildID: parsedGID,

		JoinedAt: joinedParsed,
		Nick:     member.Nick,
		Deaf:     member.Deaf,
		Mute:     member.Mute,
		Roles:    make([]int64, 0, len(member.Roles)),
	}

	for _, r := range member.Roles {
		parsed, _ := strconv.ParseInt(r, 10, 64)
		model.Roles = append(model.Roles, parsed)
	}

	err = model.Upsert(transaction, true, []string{"user_id", "guild_id"}, []string{"left_at", "joined_at", "nick", "deaf", "mute", "roles"})
	if s.handleError(err, "Failed upserting member") {
		transaction.Rollback()
		return
	}

	// args := []interface{}{parsedMID, parsedGID}
	// query := "DELETE FROM discord_member_roles WHERE user_id = $1 AND guild_id = $2"
	// if len(member.Roles) > 0 {
	// 	query += " AND role_id NOT IN ("
	// 	for i, v := range member.Roles {
	// 		if i != 0 {
	// 			query += ","
	// 		}
	// 		query += "$" + strconv.Itoa(i+3)
	// 		args = append(args, v)
	// 	}
	// 	query += ")"
	// }

	// _, err = transaction.Exec(query, args...)
	// if s.handleError(err, "Failed removing member old roles") {
	// 	transaction.Rollback()
	// 	return
	// }

	// for _, v := range member.Roles {
	// 	transaction.Exec("SAVEPOINT sp")
	// 	parsedRoleID, err := strconv.ParseInt(v, 10, 64)
	// 	if s.handleError(err, "Failed parsing role id") {
	// 		continue
	// 	}
	// 	model := models.DiscordMemberRole{
	// 		UserID:  parsedMID,
	// 		GuildID: parsedGID,
	// 		RoleID:  parsedRoleID,
	// 	}

	// 	err = model.Upsert(transaction, false, []string{"user_id", "guild_id"}, nil)
	// 	if err != nil {
	// 		if err.Error() != `pq: insert or update on table "discord_member_roles" violates foreign key constraint "discord_member_roles_role_id_fkey"` {
	// 			s.handleError(err, fmt.Sprint("Failed upserting member role update ", parsedRoleID, " : ", parsedGID))
	// 		}
	// 		transaction.Exec("ROLLBACK TO SAVEPOINT sp")
	// 	}
	// }

	err = transaction.Commit()
	s.handleError(err, "Failed committing updating member roles")
}

func (s *Server) updateRole(guildID string, role *discordgo.Role) {
	parsedGuildID, err := strconv.ParseInt(guildID, 10, 64)
	panicErr(err)

	roleIdParsed, err := strconv.ParseInt(role.ID, 10, 64)
	panicErr(err)

	model := &models.DiscordGuildRole{
		ID:      roleIdParsed,
		GuildID: parsedGuildID,

		Name:        role.Name,
		Managed:     role.Managed,
		Mentionable: role.Mentionable,
		Hoist:       role.Hoist,
		Color:       role.Color,
		Position:    role.Position,
		Permissions: role.Permissions,
	}

	err = model.Upsert(s.db, true, []string{"id"}, []string{"name", "mentionable", "hoist", "color", "position", "permissions"})
	s.handleError(err, "Failed upserting role")
}

func (s *Server) removeRole(roleID string) {
	s.db.Exec("UPDATE discord_guild_roles SET deleted_at = $2 WHERE id = $1", roleID, time.Now())
}

func (s *Server) updateGuildChannel(channel *discordgo.Channel) {
	parsedChannelId, err := strconv.ParseInt(channel.ID, 10, 64)
	panicErr(err)

	parsedGuildID, err := strconv.ParseInt(channel.GuildID, 10, 64)
	panicErr(err)

	var lastMessageID int64
	if channel.LastMessageID != "" {
		lastMessageID, _ = strconv.ParseInt(channel.LastMessageID, 10, 64)
	}

	model := &models.DiscordGuildChannel{
		ID:      parsedChannelId,
		GuildID: parsedGuildID,

		Name:          channel.Name,
		Topic:         channel.Topic,
		Type:          channel.Type,
		LastMessageID: lastMessageID,
		Position:      channel.Position,
		Bitrate:       channel.Bitrate,
	}

	err = model.Upsert(s.db, true, []string{"id"}, []string{"name", "topic", "last_message_id", "position", "bitrate"})
	s.handleError(err, "Failed upserting guild channel")

	// Update permission overwrites
	transaction, err := s.db.Begin()
	if s.handleError(err, "Failed starting a transaction to updating a permission overwrites") {
		return
	}

	args := []interface{}{parsedChannelId}
	query := "DELETE FROM discord_channel_overwrites WHERE channel_id = $1"
	if len(channel.PermissionOverwrites) > 0 {
		query += " AND id NOT IN ("
		for i, v := range channel.PermissionOverwrites {
			if i != 0 {
				query += ","
			}
			query += "$" + strconv.Itoa(i+2)
			args = append(args, v.ID)
		}
		query += ")"
	}
	_, err = transaction.Exec(query, args...)
	if s.handleError(err, "Failed removing channel permission overwrites") {
		transaction.Rollback()
		return
	}

	for _, v := range channel.PermissionOverwrites {
		parsedID, err := strconv.ParseInt(v.ID, 10, 64)
		if s.handleError(err, "Failed parsing channel persmission overwrite id") {
			continue
		}

		model := models.DiscordChannelOverwrite{
			ID:        parsedID,
			ChannelID: parsedChannelId,
			Type:      v.Type,
			Allow:     v.Allow,
			Deny:      v.Deny,
		}
		err = model.Upsert(transaction, true, []string{"channel_id", "id"}, []string{"allow", "deny"})
		s.handleError(err, "Failed upserting permission overwrite in channel")
	}

	err = transaction.Commit()
	s.handleError(err, "Failed committing updating channel permission overwrites "+strconv.Itoa(len(channel.PermissionOverwrites)))
}

func (s *Server) updatePrivateChannel(channel *discordgo.Channel) {
	err := s.updateUser(nil, channel.Recipient)
	if s.handleError(err, "Failed upserting user") {
		return
	}

	parsedChannelId, err := strconv.ParseInt(channel.ID, 10, 64)
	panicErr(err)

	parsedRecipient, err := strconv.ParseInt(channel.Recipient.ID, 10, 64)
	panicErr(err)

	var lastMessageID int64
	if channel.LastMessageID != "" {
		lastMessageID, _ = strconv.ParseInt(channel.LastMessageID, 10, 64)
	}

	model := &models.DiscordPrivateChannel{
		ID:          parsedChannelId,
		RecipientID: parsedRecipient,

		Name:          channel.Name,
		Topic:         channel.Topic,
		LastMessageID: lastMessageID,
	}

	err = model.Upsert(s.db, true, []string{"id"}, []string{"name", "topic", "last_message_id"})
	s.handleError(err, "Failed upserting private channel")
}

func (s *Server) updateVoiecState(vc *discordgo.VoiceState) {
	parsedUser, err := strconv.ParseInt(vc.UserID, 10, 64)
	panicErr(err)

	// Groups
	parsedGuildID, _ := strconv.ParseInt(vc.GuildID, 10, 64)

	if vc.ChannelID == "" {
		query := "DELETE FROM discord_voice_States WHERE user_id = $1"
		args := []interface{}{parsedUser}
		if parsedGuildID != 0 {
			query += " AND guild_id = $2"
			args = append(args, parsedGuildID)
		}

		_, err := s.db.Exec(query, args...)
		s.handleError(err, "Failed removing voice state")
		return
	}

	parsedChannelID, err := strconv.ParseInt(vc.ChannelID, 10, 64)
	panicErr(err)

	model := &models.DiscordVoiceState{
		UserID:    parsedUser,
		ChannelID: parsedChannelID,
		GuildID:   parsedGuildID,

		Surpress: vc.Suppress,
		SelfMute: vc.SelfMute,
		SelfDeaf: vc.SelfDeaf,
		Mute:     vc.Mute,
		Deaf:     vc.Deaf,
	}

	err = model.Upsert(s.db, true, []string{"guild_id", "user_id"}, []string{"surpress", "self_mute", "self_deaf", "mute", "deaf"})
	s.handleError(err, "Failed upserting voice state")
}

func (s *Server) messageCreate(m *discordgo.Message) {
	parsedMID, err := strconv.ParseInt(m.ID, 10, 64)
	if s.handleError(err, "Failed handling message create, failed parsing message id") {
		return
	}

	parsedCID, err := strconv.ParseInt(m.ChannelID, 10, 64)
	if s.handleError(err, "Failed handling message create, failed parsing channel id") {
		return
	}

	parsedTimeStamp, _ := m.Timestamp.Parse()

	transaction, err := s.db.Begin()
	if s.handleError(err, "Failed handling message create, failed starting transaction") {
		return
	}

	parsedAuthorID, _ := strconv.ParseInt(m.Author.ID, 10, 64)
	parsedAuthorDiscrim, _ := strconv.ParseInt(m.Author.Discriminator, 10, 32)

	model := &models.DiscordMessage{
		ID:        parsedMID,
		ChannelID: parsedCID,
		Timestamp: parsedTimeStamp,

		AuthorID:       parsedAuthorID,
		AuthorUsername: m.Author.Username,
		AuthorDiscrim:  int(parsedAuthorDiscrim),
		AuthorAvatar:   m.Author.Avatar,
		AuthorBot:      m.Author.Bot,

		Mentions:        []int64{},
		MentionRoles:    []int64{},
		MentionEveryone: m.MentionEveryone,

		Content: m.Content,
		Embeds:  []int64{},
	}

	err = model.Insert(transaction)
	if s.handleError(err, "Failed inserting new message") {
		transaction.Rollback()
		return
	}

	s.messageUpdate(transaction, model, m, false)
}

// Somewhat complicated update procedure:
// 1. Retry in 2 seconds if we got the update before the create
// 2. Lock the message for update
// 3. Create the revision model
// 4. create the embeds models
// 5. update the revision model
// 6. update the message model
// 7. Commit if all went well
func (s *Server) messageUpdate(transaction *sql.Tx, messageModel *models.DiscordMessage, m *discordgo.Message, retry bool) {
	parsedMID, _ := strconv.ParseInt(m.ID, 10, 64)

	if transaction == nil {
		var err error
		transaction, err = s.db.Begin()
		if s.handleError(err, "Failed updating message, failed starting transatcion") {
			return
		}

		messageModel, err = models.DiscordMessages(transaction, qm.Where("id = ?", parsedMID), qm.For("UPDATE")).One()
		if err == sql.ErrNoRows && retry {
			// Try again in a couple seconds in case of a fast embed update, or something like that
			transaction.Rollback()
			time.Sleep(time.Second * 2)
			s.messageUpdate(nil, nil, m, false)
			return
		}

		if s.handleError(err, fmt.Sprintf("Failed updating message, failed finding message: id: %d (%s)", parsedMID, m.ID)) {
			transaction.Rollback()
			return
		}
	}

	num, err := models.DiscordMessageRevisions(transaction, qm.Where("message_id = ?", parsedMID)).Count()
	if s.handleError(err, "Failed updating message, counting revisions") {
		transaction.Rollback()
		return
	}

	revisionModel := &models.DiscordMessageRevision{
		MessageID:    parsedMID,
		RevisionNum:  int(num),
		Content:      m.Content,
		Embeds:       []int64{},
		Mentions:     []int64{},
		MentionRoles: []int64{},
	}
	err = revisionModel.Insert(transaction)
	if s.handleError(err, "Failed updating message, inserting new revision") {
		transaction.Rollback()
		return
	}

	embedIds := make([]int64, 0)

	messageModel.Content = m.Content
	for _, v := range m.Embeds {
		embedmodel := createEmbedModel(v)
		embedmodel.MessageID = parsedMID
		embedmodel.RevisionNum = int(num)
		err = embedmodel.Insert(transaction)
		if s.handleError(err, "Failed updating message, inserting embed") {
			transaction.Rollback()
			return
		}

		embedIds = append(embedIds, embedmodel.ID)
	}

	revisionModel.Embeds = embedIds
	if s.handleError(revisionModel.Update(transaction, "embeds"), "Failed updating message, updating revision") {
		transaction.Rollback()
		return
	}

	messageModel.Embeds = embedIds
	parsedEdited, _ := m.EditedTimestamp.Parse()
	messageModel.EditedTimestamp = parsedEdited

	err = messageModel.Update(transaction, "content", "embeds", "edited_timestamp")
	if s.handleError(err, "Failed updating message, updating message model") {
		transaction.Rollback()
		return
	}

	if s.handleError(transaction.Commit(), "Failed updating message, comitting transation") {
		return
	}
}

func createEmbedModel(embed *discordgo.MessageEmbed) *models.DiscordMessageEmbed {
	// And here the long ass journy of creating an embed starts
	model := &models.DiscordMessageEmbed{
		URL:         embed.URL,
		Type:        embed.Type,
		Title:       embed.Title,
		Description: embed.Description,
		Timestamp:   embed.Timestamp,
		Color:       embed.Color,
	}

	model.FieldNames = make([]string, len(embed.Fields))
	model.FieldValues = make([]string, len(embed.Fields))
	model.FieldInlines = make([]bool, len(embed.Fields))
	if len(embed.Fields) > 0 {
		for i := 0; i < len(embed.Fields); i++ {
			f := embed.Fields[i]
			model.FieldNames[i] = f.Name
			model.FieldValues[i] = f.Value
			model.FieldInlines[i] = f.Inline
		}
	}

	if embed.Footer != nil {
		model.FooterText = null.StringFrom(embed.Footer.Text)
		model.FooterIconURL = null.StringFrom(embed.Footer.IconURL)
		model.FooterProxyIconURL = null.StringFrom(embed.Footer.ProxyIconURL)
	}

	if embed.Thumbnail != nil {
		model.ThumbnailURL = null.StringFrom(embed.Thumbnail.URL)
		model.ThumbnailProxyURL = null.StringFrom(embed.Thumbnail.ProxyURL)
		model.ThumbnailWidth = null.IntFrom(embed.Thumbnail.Width)
		model.ThumbnailHeight = null.IntFrom(embed.Thumbnail.Height)
	}

	if embed.Image != nil {
		model.ImageURL = null.StringFrom(embed.Image.URL)
		model.ImageProxyURL = null.StringFrom(embed.Image.ProxyURL)
		model.ImageHeight = null.IntFrom(embed.Image.Height)
		model.ImageWidth = null.IntFrom(embed.Image.Width)
	}

	if embed.Video != nil {
		model.VideoURL = null.StringFrom(embed.Video.URL)
		model.VideoProxyURL = null.StringFrom(embed.Video.ProxyURL)
		model.VideoWidth = null.IntFrom(embed.Video.Width)
		model.VideoHeight = null.IntFrom(embed.Video.Height)
	}

	if embed.Provider != nil {
		model.ProviderName = null.StringFrom(embed.Provider.Name)
		model.ProviderURL = null.StringFrom(embed.Provider.URL)
	}

	if embed.Author != nil {
		model.AuthorURL = null.StringFrom(embed.Author.URL)
		model.AuthorName = null.StringFrom(embed.Author.Name)
		model.AuthorIconURL = null.StringFrom(embed.Author.IconURL)
		model.AuthorProxyIconURL = null.StringFrom(embed.Author.ProxyIconURL)
	}

	return model
}

func (s *Server) messageDelete(m *discordgo.Message) {
	err := models.DiscordMessages(s.db, qm.Where("id = ?", m.ID)).UpdateAll(models.M{"deleted_at": time.Now()})
	s.handleError(err, "Failed marking message as deleted")
}
