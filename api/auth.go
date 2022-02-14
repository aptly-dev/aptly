package api

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"
)

func authorize(username string, password string) error {

	attributes := []string{"DN", "CN"}
	var err error

	config := context.Config()
	dn := fmt.Sprintf("%s", config.LdapDN)
	filter := fmt.Sprintf(config.LdapFilter, username)

	// connect to ldap server
	server := fmt.Sprintf("%s", config.LdapServer)
	conn, err := ldap.Dial("tcp", server)
	if err != nil {
		return err
	}
	defer conn.Close()

	// reconnect via tls
	err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return err
	}

	// format our request and then fire it off
	request := ldap.NewSearchRequest(dn, ldap.ScopeWholeSubtree, 0, 0, 0, false, filter, attributes, nil)
	search, err := conn.Search(request)
	if err != nil {
		return err
	}
	// get our modified dn and then check our user for auth
	udn := search.Entries[0].DN
	err = conn.Bind(udn, password)
	if err != nil {
		return err
	}
	return nil
}

func getGroups(c *gin.Context, username string) {

	var groups []string
	config := context.Config()
	dn := fmt.Sprintf("%s", config.LdapDN)
	session := sessions.Default(c)
	// connect to ldap server
	server := fmt.Sprintf("%s", config.LdapServer)
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
	return
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

func CheckGroup (c *gin.Context, ldgroup string) error err {
	if (!checkGroup(c, ldgroup)) {
		err = fmt.Errorf("Authorisation Failred")
	}
	return err
}
