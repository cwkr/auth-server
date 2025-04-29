package oauth2

import (
	"github.com/go-ldap/ldap/v3"
	"log"
	"slices"
	"strings"
)

type RoleMapping struct {
	ByGroup   []string `json:"by_group,omitempty"`
	ByGroupDN []string `json:"by_group_dn,omitempty"`
	ByUserID  []string `json:"by_user_id,omitempty"`
}

type RoleMappings map[string]RoleMapping

func (c RoleMappings) Roles(user User) []string {
	var roles = make([]string, 0)
	if slices.Contains(c["*"].ByGroup, "*") {
		roles = append(roles, user.Groups...)
	}
	for role, mapping := range c {
		if role == "*" {
			continue
		}
		if len(mapping.ByUserID) > 0 {
			for _, userID := range mapping.ByUserID {
				if strings.EqualFold(strings.TrimSpace(userID), user.UserID) {
					if !slices.Contains(roles, role) {
						roles = append(roles, role)
					}
					break
				}
			}
		}
		if len(mapping.ByGroupDN) > 0 {
			var found = false
			for _, groupDN := range mapping.ByGroupDN {
				var wantedDN *ldap.DN
				if dn, err := ldap.ParseDN(groupDN); err != nil {
					log.Print(err)
					break
				} else {
					wantedDN = dn
				}
				for _, group := range user.Groups {
					if dn, err := ldap.ParseDN(group); err != nil {
						log.Print(err)
						continue
					} else {
						if dn.EqualFold(wantedDN) {
							if !slices.Contains(roles, role) {
								roles = append(roles, role)
							}
							found = true
							break
						}
					}
				}
				if found {
					break
				}
			}
		}
		if len(mapping.ByGroup) > 0 {
			var found = false
			for _, wantedGroup := range mapping.ByGroup {
				for _, userGroup := range user.Groups {
					if strings.EqualFold(strings.TrimSpace(wantedGroup), userGroup) {
						if !slices.Contains(roles, role) {
							roles = append(roles, role)
						}
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
	}
	log.Printf("%s mapped roles: %v", user.UserID, roles)
	return roles
}
