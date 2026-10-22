package mwanachamacatalog_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func TestSpecForGivesEachAgencyAndMountItsOwnTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	seen := map[string]string{}
	for _, instance := range []string{"agya1b2c3d4", "agyf9e8d7c6"} {
		for _, mount := range []string{"tours", "tenders"} {
			s, err := catalog.SpecFor(instance, mount)
			if err != nil {
				t.Fatalf("SpecFor(%q, %q): %v", instance, mount, err)
			}
			if s.Instance != instance || s.MountName() != mount {
				t.Fatalf("SpecFor(%q, %q) gave instance %q mount %q", instance, mount, s.Instance, s.MountName())
			}
			if err := spec.Migrate(db, s); err != nil {
				t.Fatalf("migrate %s/%s: %v", instance, mount, err)
			}
			if _, err := catalog.NewCatalogManager(db, s); err != nil {
				t.Fatalf("manager %s/%s: %v", instance, mount, err)
			}
			for _, o := range s.Objects {
				table := s.TableFor(o)
				if prior, taken := seen[table]; taken {
					t.Fatalf("%s lands in %s, already taken by %s", s.RawNameFor(o), table, prior)
				}
				seen[table] = s.RawNameFor(o)
				if !db.Migrator().HasTable(table) {
					t.Fatalf("%s was not created", s.RawNameFor(o))
				}
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("no tables declared")
	}
}

func TestSpecForWithNoMountIsTheDefaultMount(t *testing.T) {
	s, err := catalog.SpecFor("agya1b2c3d4", "")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if s.MountName() != spec.DefaultMount {
		t.Fatalf("mount = %q, want %q", s.MountName(), spec.DefaultMount)
	}
}

func TestSpecForRefusesAMountOutsideTheAlphabet(t *testing.T) {
	if _, err := catalog.SpecFor("agya1b2c3d4", "Tours & Guides"); err == nil {
		t.Fatal("a mount with spaces and an ampersand was accepted")
	}
}

func TestSpecForLeavesTheEmbeddedDeclarationAlone(t *testing.T) {
	first, err := catalog.SpecFor("agya1b2c3d4", "tours")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	second, err := catalog.SpecFor("agyf9e8d7c6", "tenders")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if first.Instance == second.Instance || first.MountName() == second.MountName() {
		t.Fatal("a second SpecFor changed the first one's instance or mount")
	}
}
