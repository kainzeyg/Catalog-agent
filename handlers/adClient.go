package handlers

import (
	"catalog-agent/audit"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

type ADClient struct {
	config      ADConfig
	logger      *audit.Logger
	conn        *ldap.Conn
	lastConnect time.Time
}

type ADConfig struct {
	Server   string
	Port     int
	BaseDN   string
	BindUser string
	BindPass string
	Timeout  time.Duration
	Enabled  bool
}

func NewADClient(config ADConfig, logger *audit.Logger) *ADClient {
	return &ADClient{
		config: config,
		logger: logger,
	}
}

func (c *ADClient) GetUserGroups(userUPN string) ([]string, error) {
	if !c.config.Enabled || c.config.Server == "" {
		c.logger.Debug("AD disabled or not configured, returning empty groups")
		return []string{}, nil
	}

	// Extract username from UPN
	username := strings.Split(userUPN, "@")[0]
	if username == "" {
		username = userUPN
	}

	c.logger.Debug("Getting AD groups for user: %s", username)

	// Connect to AD
	if err := c.connect(); err != nil {
		c.logger.Warn("Failed to connect to AD: %v", err)
		return []string{}, nil // Return empty groups, allow all resources
	}

	// Search for user
	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(&(objectClass=user)(sAMAccountName=%s))", username),
		[]string{"memberOf", "sAMAccountName", "displayName"},
		nil,
	)

	if c.config.Timeout > 0 {
		c.conn.SetTimeout(c.config.Timeout)
	}

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		c.logger.Warn("AD search failed: %v", err)
		return []string{}, nil
	}

	if len(result.Entries) == 0 {
		c.logger.Warn("User not found in AD: %s", username)
		return []string{}, nil
	}

	// Extract groups
	groups := []string{}
	for _, attr := range result.Entries[0].Attributes {
		if attr.Name == "memberOf" {
			for _, val := range attr.Values {
				if cn := extractCN(val); cn != "" {
					groups = append(groups, cn)
				}
			}
		}
	}

	c.logger.Debug("Found %d groups for user %s", len(groups), username)
	return groups, nil
}

func (c *ADClient) connect() error {
	// Check if connection is still valid
	if c.conn != nil && time.Since(c.lastConnect) < 5*time.Minute {
		return nil
	}

	// Close old connection
	if c.conn != nil {
		c.conn.Close()
	}

	addr := fmt.Sprintf("%s:%d", c.config.Server, c.config.Port)

	var conn *ldap.Conn
	var err error

	// Connect with TLS if port is 636
	if c.config.Port == 636 {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{
			InsecureSkipVerify: false,
		})
	} else {
		conn, err = ldap.Dial("tcp", addr)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to AD: %w", err)
	}

	// Bind
	if c.config.BindUser != "" && c.config.BindPass != "" {
		err = conn.Bind(c.config.BindUser, c.config.BindPass)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to bind to AD: %w", err)
		}
	}

	c.conn = conn
	c.lastConnect = time.Now()
	return nil
}

func (c *ADClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func extractCN(dn string) string {
	parts := strings.Split(dn, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "CN=") {
			return strings.TrimPrefix(part, "CN=")
		}
	}
	return ""
}
