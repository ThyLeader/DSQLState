package models

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/vattle/sqlboiler/boil"
	"github.com/vattle/sqlboiler/randomize"
	"github.com/vattle/sqlboiler/strmangle"
)

func testDiscordChannelOverwrites(t *testing.T) {
	t.Parallel()

	query := DiscordChannelOverwrites(nil)

	if query.Query == nil {
		t.Error("expected a query, got nothing")
	}
}
func testDiscordChannelOverwritesDelete(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	if err = discordChannelOverwrite.Delete(tx); err != nil {
		t.Error(err)
	}

	count, err := DiscordChannelOverwrites(tx).Count()
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testDiscordChannelOverwritesQueryDeleteAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	if err = DiscordChannelOverwrites(tx).DeleteAll(); err != nil {
		t.Error(err)
	}

	count, err := DiscordChannelOverwrites(tx).Count()
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testDiscordChannelOverwritesSliceDeleteAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	slice := DiscordChannelOverwriteSlice{discordChannelOverwrite}

	if err = slice.DeleteAll(tx); err != nil {
		t.Error(err)
	}

	count, err := DiscordChannelOverwrites(tx).Count()
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}
func testDiscordChannelOverwritesExists(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	e, err := DiscordChannelOverwriteExists(tx, discordChannelOverwrite.ID, discordChannelOverwrite.ChannelID)
	if err != nil {
		t.Errorf("Unable to check if DiscordChannelOverwrite exists: %s", err)
	}
	if !e {
		t.Errorf("Expected DiscordChannelOverwriteExistsG to return true, but got false.")
	}
}
func testDiscordChannelOverwritesFind(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	discordChannelOverwriteFound, err := FindDiscordChannelOverwrite(tx, discordChannelOverwrite.ID, discordChannelOverwrite.ChannelID)
	if err != nil {
		t.Error(err)
	}

	if discordChannelOverwriteFound == nil {
		t.Error("want a record, got nil")
	}
}
func testDiscordChannelOverwritesBind(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	if err = DiscordChannelOverwrites(tx).Bind(discordChannelOverwrite); err != nil {
		t.Error(err)
	}
}

func testDiscordChannelOverwritesOne(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	if x, err := DiscordChannelOverwrites(tx).One(); err != nil {
		t.Error(err)
	} else if x == nil {
		t.Error("expected to get a non nil record")
	}
}

func testDiscordChannelOverwritesAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwriteOne := &DiscordChannelOverwrite{}
	discordChannelOverwriteTwo := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwriteOne, discordChannelOverwriteDBTypes, false, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}
	if err = randomize.Struct(seed, discordChannelOverwriteTwo, discordChannelOverwriteDBTypes, false, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwriteOne.Insert(tx); err != nil {
		t.Error(err)
	}
	if err = discordChannelOverwriteTwo.Insert(tx); err != nil {
		t.Error(err)
	}

	slice, err := DiscordChannelOverwrites(tx).All()
	if err != nil {
		t.Error(err)
	}

	if len(slice) != 2 {
		t.Error("want 2 records, got:", len(slice))
	}
}

func testDiscordChannelOverwritesCount(t *testing.T) {
	t.Parallel()

	var err error
	seed := randomize.NewSeed()
	discordChannelOverwriteOne := &DiscordChannelOverwrite{}
	discordChannelOverwriteTwo := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwriteOne, discordChannelOverwriteDBTypes, false, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}
	if err = randomize.Struct(seed, discordChannelOverwriteTwo, discordChannelOverwriteDBTypes, false, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwriteOne.Insert(tx); err != nil {
		t.Error(err)
	}
	if err = discordChannelOverwriteTwo.Insert(tx); err != nil {
		t.Error(err)
	}

	count, err := DiscordChannelOverwrites(tx).Count()
	if err != nil {
		t.Error(err)
	}

	if count != 2 {
		t.Error("want 2 records, got:", count)
	}
}

func testDiscordChannelOverwritesInsert(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	count, err := DiscordChannelOverwrites(tx).Count()
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}
}

func testDiscordChannelOverwritesInsertWhitelist(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx, discordChannelOverwriteColumns...); err != nil {
		t.Error(err)
	}

	count, err := DiscordChannelOverwrites(tx).Count()
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}
}

func testDiscordChannelOverwritesReload(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	if err = discordChannelOverwrite.Reload(tx); err != nil {
		t.Error(err)
	}
}

func testDiscordChannelOverwritesReloadAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	slice := DiscordChannelOverwriteSlice{discordChannelOverwrite}

	if err = slice.ReloadAll(tx); err != nil {
		t.Error(err)
	}
}
func testDiscordChannelOverwritesSelect(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	slice, err := DiscordChannelOverwrites(tx).All()
	if err != nil {
		t.Error(err)
	}

	if len(slice) != 1 {
		t.Error("want one record, got:", len(slice))
	}
}

var (
	discordChannelOverwriteDBTypes = map[string]string{`Allow`: `integer`, `ChannelID`: `bigint`, `Deny`: `integer`, `ID`: `bigint`, `Type`: `character varying`}
	_                              = bytes.MinRead
)

func testDiscordChannelOverwritesUpdate(t *testing.T) {
	t.Parallel()

	if len(discordChannelOverwriteColumns) == len(discordChannelOverwritePrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	count, err := DiscordChannelOverwrites(tx).Count()
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}

	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true, discordChannelOverwriteColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	if err = discordChannelOverwrite.Update(tx); err != nil {
		t.Error(err)
	}
}

func testDiscordChannelOverwritesSliceUpdateAll(t *testing.T) {
	t.Parallel()

	if len(discordChannelOverwriteColumns) == len(discordChannelOverwritePrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	discordChannelOverwrite := &DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Insert(tx); err != nil {
		t.Error(err)
	}

	count, err := DiscordChannelOverwrites(tx).Count()
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}

	if err = randomize.Struct(seed, discordChannelOverwrite, discordChannelOverwriteDBTypes, true, discordChannelOverwritePrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	// Remove Primary keys and unique columns from what we plan to update
	var fields []string
	if strmangle.StringSliceMatch(discordChannelOverwriteColumns, discordChannelOverwritePrimaryKeyColumns) {
		fields = discordChannelOverwriteColumns
	} else {
		fields = strmangle.SetComplement(
			discordChannelOverwriteColumns,
			discordChannelOverwritePrimaryKeyColumns,
		)
	}

	value := reflect.Indirect(reflect.ValueOf(discordChannelOverwrite))
	updateMap := M{}
	for _, col := range fields {
		updateMap[col] = value.FieldByName(strmangle.TitleCase(col)).Interface()
	}

	slice := DiscordChannelOverwriteSlice{discordChannelOverwrite}
	if err = slice.UpdateAll(tx, updateMap); err != nil {
		t.Error(err)
	}
}
func testDiscordChannelOverwritesUpsert(t *testing.T) {
	t.Parallel()

	if len(discordChannelOverwriteColumns) == len(discordChannelOverwritePrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	// Attempt the INSERT side of an UPSERT
	discordChannelOverwrite := DiscordChannelOverwrite{}
	if err = randomize.Struct(seed, &discordChannelOverwrite, discordChannelOverwriteDBTypes, true); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	tx := MustTx(boil.Begin())
	defer tx.Rollback()
	if err = discordChannelOverwrite.Upsert(tx, false, nil, nil); err != nil {
		t.Errorf("Unable to upsert DiscordChannelOverwrite: %s", err)
	}

	count, err := DiscordChannelOverwrites(tx).Count()
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Error("want one record, got:", count)
	}

	// Attempt the UPDATE side of an UPSERT
	if err = randomize.Struct(seed, &discordChannelOverwrite, discordChannelOverwriteDBTypes, false, discordChannelOverwritePrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize DiscordChannelOverwrite struct: %s", err)
	}

	if err = discordChannelOverwrite.Upsert(tx, true, nil, nil); err != nil {
		t.Errorf("Unable to upsert DiscordChannelOverwrite: %s", err)
	}

	count, err = DiscordChannelOverwrites(tx).Count()
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Error("want one record, got:", count)
	}
}