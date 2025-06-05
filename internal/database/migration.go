package database

import (
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"

	"gorm.io/gorm"
)

//go:embed migrations/*/up.sql migrations/*/down.sql
var migrationsFS embed.FS

type SchemaVersion uint64

type SchemaMigration struct {
	Version SchemaVersion `gorm:"primaryKey"`
}

func CurrentSchemaVersion(db *gorm.DB) SchemaVersion {
	return CurrentSchemaMigration(db).Version
}

func CurrentSchemaMigration(db *gorm.DB) SchemaMigration {
	var schemaMigration SchemaMigration

	db.
		Model(&SchemaMigration{}).
		Select("version").
		Order("version desc").
		Limit(1).
		Scan(&schemaMigration)

	return schemaMigration
}

type Migration struct {
	Version SchemaVersion
	Dir     fs.DirEntry
}

func (migration *Migration) exec(db *gorm.DB, sql string) error {
	result := db.Exec(sql)
	if result.Error != nil {
		db.Rollback()
		return result.Error
	}

	return nil
}

func (migration *Migration) Up(db *gorm.DB) error {
	sql, err := migration.UpSQL()
	if err != nil {
		return err
	}

	return migration.exec(db, sql)
}

func (migration *Migration) Down(db *gorm.DB) error {
	sql, err := migration.DownSQL()
	if err != nil {
		return err
	}

	return migration.exec(db, sql)
}

func (migration *Migration) UpSQL() (string, error) {
	upSQL, err := fs.ReadFile(migrationsFS, fmt.Sprintf("migrations/%s/up.sql", migration.DirName()))
	if err != nil {
		return "", fmt.Errorf("failed to read up.sql for migration %s: %w", migration.DirName(), err)
	}

	return string(upSQL), nil
}

func (migration *Migration) DownSQL() (string, error) {
	downSQL, err := fs.ReadFile(
		migrationsFS,
		fmt.Sprintf(
			"migrations/%s/down.sql",
			migration.DirName(),
		),
	)
	if err != nil {
		return "", fmt.Errorf("failed to read down.sql for migration %s: %w", migration.DirName(), err)
	}

	return string(downSQL), nil
}

func (migration *Migration) DirName() string {
	return migration.Dir.Name()
}

func Migrate(db *gorm.DB) error {
	db.AutoMigrate(&SchemaMigration{})

	currentVersion := CurrentSchemaVersion(db)
	migrations, err := MigrationsNewerThan(currentVersion)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		err := db.Transaction(func(tx *gorm.DB) error {
			schemaMigration := SchemaMigration{
				Version: migration.Version,
			}

			tx.Create(&schemaMigration)

			return migration.Up(tx)
		})
		if err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

func MigrationsNewerThan(minVersion SchemaVersion) ([]Migration, error) {
	migrationVersionRegex := regexp.MustCompile(`^(\d+)`)

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}

	var migrations []Migration
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		match := migrationVersionRegex.FindStringSubmatch(entry.Name())

		if len(match) != 2 {
			return nil, fmt.Errorf("invalid migration directory name: %s - contains more than one version number", entry.Name())
		}

		versionInt, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid migration version: %s - %w", match[1], err)
		}

		version := SchemaVersion(versionInt)

		if version <= minVersion {
			continue
		}

		migration := Migration{
			Version: version,
			Dir:     entry,
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}
