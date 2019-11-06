package discordrolemanager

import (
	"errors"
	"fmt"

	"github.com/auttaja/discordgo"
	"github.com/casbin/casbin/rbac"
)

type RoleManager struct {
	session *discordgo.Session
}

func NewRoleManager(s *discordgo.Session) rbac.RoleManager {
	rm := RoleManager{session: s}

	return &rm
}

func (rm *RoleManager) Clear() error {
	return nil
}

// AddLink adds the inheritance link between role: name1 and role: name2.
func (rm *RoleManager) AddLink(name1 string, name2 string, domain ...string) error {
	return errors.New("not implemented")
}

// DeleteLink deletes the inheritance link between role: name1 and role: name2.
func (rm *RoleManager) DeleteLink(name1 string, name2 string, domain ...string) error {
	return errors.New("not implemented")
}

func (rm *RoleManager) HasLink(name1 string, name2 string, domain ...string) (bool, error) {
	if len(domain) < 1 {
		return false, errors.New("error: domain required")
	}

	fmt.Printf("Checking link between %s and %s in %s\n", name1, name2, domain[0])

	if name1 == "" || name2 == "" {
		return false, nil
	}

	if name1 == name2 {
		return true, nil
	}

	roles, err := rm.GetRoles(name1, domain[0])
	if err != nil {
		return false, err
	}

	for _, role := range roles {
		if role == name2 {
			return true, nil
		}
	}

	return false, nil
}

func (rm *RoleManager) GetRoles(name string, domain ...string) ([]string, error) {
	if len(domain) < 1 {
		return nil, errors.New("error: domain required")
	}

	fmt.Printf("Getting roles for %s in %s\n", name, domain[0])

	m, err := rm.session.State.Member(domain[0], name)
	if err != nil {
		m, err = rm.session.GuildMember(domain[0], name)
		if err != nil {
			return nil, err
		}
	}

	return m.Roles, nil
}

func (rm *RoleManager) GetUsers(name string, domain ...string) ([]string, error) {
	if len(domain) < 1 {
		return nil, errors.New("error: domain required")
	}

	fmt.Printf("Getting users for role %s in %s\n", name, domain[0])

	g, err := rm.session.State.Guild(domain[0])
	if err != nil {
		g, err = rm.session.Guild(domain[0])
		if err != nil {
			return nil, err
		}
	}

	var users []string

	for i := 0; i < len(g.Members); i++ {
		for j := 0; j < len(g.Members[i].Roles); j++ {
			if g.Members[i].Roles[j] == name {
				users = append(users, g.Members[i].User.ID)
				break
			}
		}
	}

	return users, nil
}

// PrintRoles prints all the roles to log.
func (rm *RoleManager) PrintRoles() error {
	return errors.New("not implemented")
}
