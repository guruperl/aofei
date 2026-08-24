package acl

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDBGetACLAudienceContextCanceledDoesNotQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := DBGetACLAudienceContext(ctx, db, 33); !errors.Is(err, context.Canceled) {
		t.Fatalf("DBGetACLAudienceContext error = %v, want context canceled", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDBGetPubAppAudienceInheritanceMatrix(t *testing.T) {
	for _, advertiserOrder := range []string{"White", "Black"} {
		for _, campaignOrder := range []string{"Inherit", "White", "Black"} {
			for _, itemOrder := range []string{"Inherit", "White", "Black"} {
				name := fmt.Sprintf("advertiser_%s/campaign_%s/item_%s", advertiserOrder, campaignOrder, itemOrder)
				t.Run(name, func(t *testing.T) {
					db, mock, err := sqlmock.New()
					if err != nil {
						t.Fatal(err)
					}
					defer db.Close()

					const advertiserID uint32 = 11
					const campaignID uint32 = 22
					const itemID uint32 = 33
					mock.ExpectQuery(`(?s)SELECT a\.domain.*WHERE i\.item_id=\?`).
						WithArgs(itemID).
						WillReturnRows(sqlmock.NewRows([]string{
							"domain", "foreign_id", "adv_id", "adv_order", "campaign_id", "campaign_order", "site_types", "item_order",
						}).AddRow("advertiser.example", "campaign-external", advertiserID, advertiserOrder, campaignID, campaignOrder, "Web", itemOrder))

					effectiveOrder := itemOrder
					effectiveID := itemID
					entityType := 42
					if itemOrder == "Inherit" {
						effectiveOrder = campaignOrder
						effectiveID = campaignID
						entityType = 41
						if campaignOrder == "Inherit" {
							effectiveOrder = advertiserOrder
							effectiveID = advertiserID
							entityType = 4
						}
					}

					if itemOrder == "Inherit" {
						publisherPattern := fmt.Sprintf(`(?s)SELECT p\.domain.*WHERE .*ac\.entitytype_id=%d AND ac\.entity_id=\?`, entityType)
						if entityType == 41 {
							publisherPattern += `\)`
						}
						mock.ExpectQuery(publisherPattern).
							WithArgs(effectiveID).
							WillReturnRows(sqlmock.NewRows([]string{"domain"}).AddRow("publisher.example"))
					}

					appPattern := fmt.Sprintf(`(?s)SELECT .*foreign_id.*WHERE .*ac\.entitytype_id=%d AND ac\.entity_id=\?`, entityType)
					mock.ExpectQuery(appPattern).
						WithArgs(effectiveID).
						WillReturnRows(sqlmock.NewRows([]string{"foreign_id"}).AddRow("site.example"))

					audience := new(ACLAudience)
					if err := dbGetPubAppAudience(db, itemID, audience); err != nil {
						t.Fatal(err)
					}
					if audience.SiteTypes != SiteTypeWeb {
						t.Fatalf("site type = %d, want Web", audience.SiteTypes)
					}
					if effectiveOrder == "White" {
						if itemOrder == "Inherit" && len(audience.WPub) != 1 {
							t.Fatalf("white publishers = %#v, want one", audience.WPub)
						}
						if len(audience.WApp) != 1 || audience.WApp[0] != "site.example" || audience.BPub != nil || audience.BApp != nil {
							t.Fatalf("white audience = %+v", audience)
						}
					} else {
						if itemOrder == "Inherit" && len(audience.BPub) != 1 {
							t.Fatalf("black publishers = %#v, want one", audience.BPub)
						}
						if len(audience.BApp) != 1 || audience.BApp[0] != "site.example" || audience.WPub != nil || audience.WApp != nil {
							t.Fatalf("black audience = %+v", audience)
						}
					}
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestACLAudienceHasNilInputsFailSafely(t *testing.T) {
	var nilAudience *ACLAudience
	if nilAudience.Has(&ACL{}) {
		t.Fatal("nil audience matched")
	}
	if (&ACLAudience{}).Has(nil) {
		t.Fatal("nil ACL matched")
	}
	if !(&ACLAudience{}).Has(&ACL{}) {
		t.Fatal("empty audience should remain a wildcard for a valid ACL")
	}
}
