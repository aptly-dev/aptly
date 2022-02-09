package api

import (
	"crypto/tls"
	"fmt"

	"github.com/mavricknz/ldap"
)

func Authorize(username string, password string) error {

	var attributes []string
	var err error

	config := context.Config()
	dn := fmt.Sprintf("%s", config.LdapDN)
	filter := fmt.Sprintf("%s", config.LdapFilter)

	server := fmt.Sprintf("%s", config.LdapServer)

	// setup to ignore insecure TLS, this needs to be paramaterized in the future
	tls := &tls.Config{InsecureSkipVerify: true}
	conn := &ldap.LDAPConnection{Addr: server,
		IsSSL:     true,
		TlsConfig: tls,
	}
	err = conn.Connect()
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	request := ldap.NewSearchRequest(dn, 2, 3, 0, 0, false, filter, attributes, nil)
	search, err := conn.Search(request)
	if err != nil {
		return err
	}
	if len(search.Entries) < 1 {
		return err
	}
	udn := search.Entries[0].DN
	err = conn.Bind(udn, password)
	if err != nil {
		return err
	}
	return nil

}
