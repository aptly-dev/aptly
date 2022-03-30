package api

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"
)

func Authorize(username string, password string) (ok bool) {
	config := context.Config()

	if config.Auth.Type != "" {
		switch strings.ToLower(config.Auth.Type) {
		case "ldap":
			ok = doLdapAuth(username, password)
		default:
			return false
		}
		if !ok  {
			return false
		}
	}
	return true
}

func doLdapAuth(username string, password string) bool {
	config := context.Config()
	attributes := []string{"DN", "CN"}

	server := config.Auth.Server
	dn := config.Auth.LdapDN
	filter := fmt.Sprintf(config.Auth.LdapFilter, username)

	// connect to ldap server
	conn, err := ldap.Dial("tcp", server)
	if err != nil {
		return false
	}
	defer conn.Close()

	// reconnect via tls
	err = conn.StartTLS(&tls.Config{InsecureSkipVerify: config.Auth.SecureTLS})
	if err != nil {
		return false
	}

	// format our request and then fire it off
	request := ldap.NewSearchRequest(dn, ldap.ScopeWholeSubtree, 0, 0, 0, false, filter, attributes, nil)
	search, err := conn.Search(request)
	if err != nil {
		return false
	}
	// get our modified dn and then check our user for auth
	udn := search.Entries[0].DN
	err = conn.Bind(udn, password)
	if err != nil {
		return false
	}
	return true
}

func getGroups(c *gin.Context, username string) {

	var groups []string
	config := context.Config()
	dn := config.Auth.LdapDN
	session := sessions.Default(c)
	// connect to ldap server
	server := fmt.Sprintf("%s", config.Auth.Server)
	conn, err := ldap.Dial("tcp", server)
	if err != nil {
		return
	}
	// reconnect via tls
	err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	filter := fmt.Sprintf("(|(member=uid=%s,ou=people,dc=llnw,dc=com)(member=uid=%s,ou=people,dc=llnw,dc=com))", username, username)
	request := ldap.NewSearchRequest(dn, ldap.ScopeWholeSubtree, 0, 0, 0, false, filter, []string{"dn", "cn"}, nil)
	search, err := conn.Search(request)
	if err != nil {
		return
	}
	if len(search.Entries) < 1 {
		return
	}
	for _, v := range search.Entries {
		value := strings.Split(strings.TrimLeft(v.DN, "cn="), ",")[0]
		groups = append(groups, fmt.Sprintf("%s,", value))
	}
	session.Set("Groups", groups)
}

func checkGroup(c *gin.Context, ldgroup string) bool {
	session := sessions.Default(c)
	groups := session.Get("Groups")
	if ldgroup == "" {
		return true
	}
	for _, v := range groups.([]string) {
		if strings.Contains(v, ldgroup) {
			return true
		}
	}
	return false
}

func CheckGroup(c *gin.Context, ldgroup string) (err error) {
	if !checkGroup(c, ldgroup) {
		err = fmt.Errorf("Authorisation Failred")
	}
	return err
}
