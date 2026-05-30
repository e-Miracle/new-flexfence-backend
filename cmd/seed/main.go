package main

import (
	"log"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/config"
	"github.com/flexfence/flexfence-backend/internal/db"
	"github.com/flexfence/flexfence-backend/internal/domain"
	mysqlstore "github.com/flexfence/flexfence-backend/internal/store/mysql"
)

func main() {
	cfg := config.Load()

	gormDB, err := db.ConnectMySQL(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	if err := db.AutoMigrate(gormDB); err != nil {
		log.Fatalf("db migration failed: %v", err)
	}

	identity := mysqlstore.NewIdentityStore(gormDB)
	data := mysqlstore.New(gormDB)

	const (
		orgSlug      = "acme-events"
		ownerEmail   = "owner@acme.test"
		ownerPass    = "changeme"
		eventTitle   = "Sample Conference"
		eventDesc    = "Seeded event for local development"
		fenceName    = "Main Hall"
	)

	if _, ok, err := identity.GetBusinessAuthByEmail(ownerEmail); err != nil {
		log.Fatalf("lookup owner failed: %v", err)
	} else if ok {
		log.Printf("seed skipped: owner %s already exists", ownerEmail)
		return
	}

	org, err := identity.CreateOrganization("Acme Events", orgSlug, "trial")
	if err != nil {
		log.Fatalf("create organization failed: %v", err)
	}

	hash, err := auth.HashPassword(ownerPass)
	if err != nil {
		log.Fatalf("hash password failed: %v", err)
	}

	owner, err := identity.CreateBusinessUser(org.ID, ownerEmail, hash, "Ada", "Owner", domain.BusinessRoleOwner)
	if err != nil {
		log.Fatalf("create business user failed: %v", err)
	}

	now := time.Now().UTC()
	eventStart := now.Add(24 * time.Hour).Truncate(time.Hour)
	eventEnd := eventStart.Add(8 * time.Hour)
	event, err := data.CreateEvent(org.ID, owner.ID, eventTitle, eventDesc, eventStart, eventEnd)
	if err != nil {
		log.Fatalf("create event failed: %v", err)
	}

	_, err = data.AddFence(event.ID, domain.FenceCreateInput{
		Name:      fenceName,
		ShapeType: "circle",
		StartAt:   eventStart,
		EndAt:     eventEnd,
		CenterLat: 6.5244,
		CenterLng: 3.3792,
		RadiusM:   120,
	})
	if err != nil {
		log.Fatalf("create fence failed: %v", err)
	}

	log.Printf("seed complete")
	log.Printf("organization_id=%s slug=%s", org.ID, org.Slug)
	log.Printf("business_user_id=%s email=%s password=%s", owner.ID, ownerEmail, ownerPass)
	log.Printf("event_id=%s", event.ID)
}
