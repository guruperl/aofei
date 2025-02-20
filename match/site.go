package match

import (
	"database/sql"
	"net/url"
	"strings"

	"github.com/genelet/winter/pzutil"
)

// Site definistion. Changes of referers and channels should refresh site cache
type Site struct {
	SiteID     uint32
	PubID      uint32
	Company    string
	SiteName   string
	SiteURL    string
	Referers   []string
	ChannelIds []uint16
}

func (self *Site) Pack() ([]byte, error) {
	return pzutil.PackObject(self)
}

func UnpackSite(data []byte) (*Site, error) {
	site := new(Site)
	err := pzutil.UnpackObject(data, site)
	return site, err
}

func (self *Site) DomainMatch(u *url.URL) bool {
	if u == nil || u.Host == "" {
		return true
	}

	if self.SiteURL != "" {
		own, err := url.Parse(self.SiteURL)
		if err == nil && u.Host == own.Host {
			return true
		}
	}

	if len(self.Referers) < 1 {
		return false
	}

	lenhost := len(u.Host)
	lenpath := len(u.Path)

	for _, domain := range self.Referers {
		parts := strings.SplitN(domain, "/", 2)
		len0 := len(parts[0])
		if len0 > lenhost {
			continue
		}
		if u.Host[lenhost-len0:] == parts[0] && (len0 == lenhost || u.Host[lenhost-len0-1] == '.') {
			if len(parts) > 1 {
				path := "/" + parts[1]
				len1 := len(path)
				if len1 > lenpath {
					continue
				}
				if u.Path[:len1] == path {
					return true
				}
			} else {
				return true
			}
		}
	}

	return false
}

func DBGetSite(db *sql.DB, siteID uint32) (*Site, error) {
	site := new(Site)
	err := db.QueryRow(`
SELECT p.pub_id, a.company, s.site_id, s.site_name, s.site_url
FROM pub_site s
INNER JOIN pub p USING (pub_id)
INNER JOIN add_address a USING (address_id)
WHERE s.site_id=?`, siteID).Scan(&site.PubID, &site.Company, &site.SiteID, &site.SiteName, &site.SiteURL)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT referer
FROM pub_referer
WHERE site_id=?`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var referer string
		err = rows.Scan(&referer)
		if err != nil {
			return nil, err
		}
		site.Referers = append(site.Referers, referer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.Query(`
SELECT channel_id
FROM ch_belong
WHERE entitytype_id=31 AND entity_id=?`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var channelID uint16
		err = rows.Scan(&channelID)
		if err != nil {
			return nil, err
		}
		site.ChannelIds = append(site.ChannelIds, channelID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return site, nil
}
