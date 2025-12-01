package onvif

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/0x524a/onvif-go/internal/soap"
)

// GetRemoteUser returns the configured remote user
func (c *Client) GetRemoteUser(ctx context.Context) (*RemoteUser, error) {
	type GetRemoteUser struct {
		XMLName xml.Name `xml:"tds:GetRemoteUser"`
		Xmlns   string   `xml:"xmlns:tds,attr"`
	}

	type GetRemoteUserResponse struct {
		XMLName    xml.Name `xml:"GetRemoteUserResponse"`
		RemoteUser *struct {
			Username           string `xml:"Username"`
			Password           string `xml:"Password"`
			UseDerivedPassword bool   `xml:"UseDerivedPassword"`
		} `xml:"RemoteUser"`
	}

	req := GetRemoteUser{
		Xmlns: deviceNamespace,
	}

	var resp GetRemoteUserResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetRemoteUser failed: %w", err)
	}

	if resp.RemoteUser == nil {
		return nil, nil
	}

	return &RemoteUser{
		Username:           resp.RemoteUser.Username,
		Password:           resp.RemoteUser.Password,
		UseDerivedPassword: resp.RemoteUser.UseDerivedPassword,
	}, nil
}

// SetRemoteUser sets the remote user
func (c *Client) SetRemoteUser(ctx context.Context, remoteUser *RemoteUser) error {
	type SetRemoteUser struct {
		XMLName    xml.Name `xml:"tds:SetRemoteUser"`
		Xmlns      string   `xml:"xmlns:tds,attr"`
		RemoteUser *struct {
			Username           string `xml:"tds:Username"`
			Password           string `xml:"tds:Password,omitempty"`
			UseDerivedPassword bool   `xml:"tds:UseDerivedPassword"`
		} `xml:"tds:RemoteUser,omitempty"`
	}

	req := SetRemoteUser{
		Xmlns: deviceNamespace,
	}

	if remoteUser != nil {
		req.RemoteUser = &struct {
			Username           string `xml:"tds:Username"`
			Password           string `xml:"tds:Password,omitempty"`
			UseDerivedPassword bool   `xml:"tds:UseDerivedPassword"`
		}{
			Username:           remoteUser.Username,
			Password:           remoteUser.Password,
			UseDerivedPassword: remoteUser.UseDerivedPassword,
		}
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetRemoteUser failed: %w", err)
	}

	return nil
}

// GetIPAddressFilter gets the IP address filter settings from a device
func (c *Client) GetIPAddressFilter(ctx context.Context) (*IPAddressFilter, error) {
	type GetIPAddressFilter struct {
		XMLName xml.Name `xml:"tds:GetIPAddressFilter"`
		Xmlns   string   `xml:"xmlns:tds,attr"`
	}

	type GetIPAddressFilterResponse struct {
		XMLName         xml.Name `xml:"GetIPAddressFilterResponse"`
		IPAddressFilter struct {
			Type        string `xml:"Type"`
			IPv4Address []struct {
				Address      string `xml:"Address"`
				PrefixLength int    `xml:"PrefixLength"`
			} `xml:"IPv4Address"`
			IPv6Address []struct {
				Address      string `xml:"Address"`
				PrefixLength int    `xml:"PrefixLength"`
			} `xml:"IPv6Address"`
		} `xml:"IPAddressFilter"`
	}

	req := GetIPAddressFilter{
		Xmlns: deviceNamespace,
	}

	var resp GetIPAddressFilterResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetIPAddressFilter failed: %w", err)
	}

	filter := &IPAddressFilter{
		Type: IPAddressFilterType(resp.IPAddressFilter.Type),
	}

	for _, addr := range resp.IPAddressFilter.IPv4Address {
		filter.IPv4Address = append(filter.IPv4Address, PrefixedIPv4Address{
			Address:      addr.Address,
			PrefixLength: addr.PrefixLength,
		})
	}

	for _, addr := range resp.IPAddressFilter.IPv6Address {
		filter.IPv6Address = append(filter.IPv6Address, PrefixedIPv6Address{
			Address:      addr.Address,
			PrefixLength: addr.PrefixLength,
		})
	}

	return filter, nil
}

// SetIPAddressFilter sets the IP address filter settings on a device
func (c *Client) SetIPAddressFilter(ctx context.Context, filter *IPAddressFilter) error {
	type SetIPAddressFilter struct {
		XMLName         xml.Name `xml:"tds:SetIPAddressFilter"`
		Xmlns           string   `xml:"xmlns:tds,attr"`
		IPAddressFilter struct {
			Type        string `xml:"tds:Type"`
			IPv4Address []struct {
				Address      string `xml:"tds:Address"`
				PrefixLength int    `xml:"tds:PrefixLength"`
			} `xml:"tds:IPv4Address,omitempty"`
			IPv6Address []struct {
				Address      string `xml:"tds:Address"`
				PrefixLength int    `xml:"tds:PrefixLength"`
			} `xml:"tds:IPv6Address,omitempty"`
		} `xml:"tds:IPAddressFilter"`
	}

	req := SetIPAddressFilter{
		Xmlns: deviceNamespace,
	}
	req.IPAddressFilter.Type = string(filter.Type)

	for _, addr := range filter.IPv4Address {
		req.IPAddressFilter.IPv4Address = append(req.IPAddressFilter.IPv4Address, struct {
			Address      string `xml:"tds:Address"`
			PrefixLength int    `xml:"tds:PrefixLength"`
		}{
			Address:      addr.Address,
			PrefixLength: addr.PrefixLength,
		})
	}

	for _, addr := range filter.IPv6Address {
		req.IPAddressFilter.IPv6Address = append(req.IPAddressFilter.IPv6Address, struct {
			Address      string `xml:"tds:Address"`
			PrefixLength int    `xml:"tds:PrefixLength"`
		}{
			Address:      addr.Address,
			PrefixLength: addr.PrefixLength,
		})
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetIPAddressFilter failed: %w", err)
	}

	return nil
}

// AddIPAddressFilter adds an IP filter address to a device
func (c *Client) AddIPAddressFilter(ctx context.Context, filter *IPAddressFilter) error {
	type AddIPAddressFilter struct {
		XMLName         xml.Name `xml:"tds:AddIPAddressFilter"`
		Xmlns           string   `xml:"xmlns:tds,attr"`
		IPAddressFilter struct {
			Type        string `xml:"tds:Type"`
			IPv4Address []struct {
				Address      string `xml:"tds:Address"`
				PrefixLength int    `xml:"tds:PrefixLength"`
			} `xml:"tds:IPv4Address,omitempty"`
			IPv6Address []struct {
				Address      string `xml:"tds:Address"`
				PrefixLength int    `xml:"tds:PrefixLength"`
			} `xml:"tds:IPv6Address,omitempty"`
		} `xml:"tds:IPAddressFilter"`
	}

	req := AddIPAddressFilter{
		Xmlns: deviceNamespace,
	}
	req.IPAddressFilter.Type = string(filter.Type)

	for _, addr := range filter.IPv4Address {
		req.IPAddressFilter.IPv4Address = append(req.IPAddressFilter.IPv4Address, struct {
			Address      string `xml:"tds:Address"`
			PrefixLength int    `xml:"tds:PrefixLength"`
		}{
			Address:      addr.Address,
			PrefixLength: addr.PrefixLength,
		})
	}

	for _, addr := range filter.IPv6Address {
		req.IPAddressFilter.IPv6Address = append(req.IPAddressFilter.IPv6Address, struct {
			Address      string `xml:"tds:Address"`
			PrefixLength int    `xml:"tds:PrefixLength"`
		}{
			Address:      addr.Address,
			PrefixLength: addr.PrefixLength,
		})
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, nil); err != nil {
		return fmt.Errorf("AddIPAddressFilter failed: %w", err)
	}

	return nil
}

// RemoveIPAddressFilter deletes an IP filter address from a device
func (c *Client) RemoveIPAddressFilter(ctx context.Context, filter *IPAddressFilter) error {
	type RemoveIPAddressFilter struct {
		XMLName         xml.Name `xml:"tds:RemoveIPAddressFilter"`
		Xmlns           string   `xml:"xmlns:tds,attr"`
		IPAddressFilter struct {
			Type        string `xml:"tds:Type"`
			IPv4Address []struct {
				Address      string `xml:"tds:Address"`
				PrefixLength int    `xml:"tds:PrefixLength"`
			} `xml:"tds:IPv4Address,omitempty"`
			IPv6Address []struct {
				Address      string `xml:"tds:Address"`
				PrefixLength int    `xml:"tds:PrefixLength"`
			} `xml:"tds:IPv6Address,omitempty"`
		} `xml:"tds:IPAddressFilter"`
	}

	req := RemoveIPAddressFilter{
		Xmlns: deviceNamespace,
	}
	req.IPAddressFilter.Type = string(filter.Type)

	for _, addr := range filter.IPv4Address {
		req.IPAddressFilter.IPv4Address = append(req.IPAddressFilter.IPv4Address, struct {
			Address      string `xml:"tds:Address"`
			PrefixLength int    `xml:"tds:PrefixLength"`
		}{
			Address:      addr.Address,
			PrefixLength: addr.PrefixLength,
		})
	}

	for _, addr := range filter.IPv6Address {
		req.IPAddressFilter.IPv6Address = append(req.IPAddressFilter.IPv6Address, struct {
			Address      string `xml:"tds:Address"`
			PrefixLength int    `xml:"tds:PrefixLength"`
		}{
			Address:      addr.Address,
			PrefixLength: addr.PrefixLength,
		})
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, nil); err != nil {
		return fmt.Errorf("RemoveIPAddressFilter failed: %w", err)
	}

	return nil
}

// GetZeroConfiguration gets the zero-configuration from a device
func (c *Client) GetZeroConfiguration(ctx context.Context) (*NetworkZeroConfiguration, error) {
	type GetZeroConfiguration struct {
		XMLName xml.Name `xml:"tds:GetZeroConfiguration"`
		Xmlns   string   `xml:"xmlns:tds,attr"`
	}

	type GetZeroConfigurationResponse struct {
		XMLName           xml.Name `xml:"GetZeroConfigurationResponse"`
		ZeroConfiguration struct {
			InterfaceToken string   `xml:"InterfaceToken"`
			Enabled        bool     `xml:"Enabled"`
			Addresses      []string `xml:"Addresses"`
		} `xml:"ZeroConfiguration"`
	}

	req := GetZeroConfiguration{
		Xmlns: deviceNamespace,
	}

	var resp GetZeroConfigurationResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetZeroConfiguration failed: %w", err)
	}

	return &NetworkZeroConfiguration{
		InterfaceToken: resp.ZeroConfiguration.InterfaceToken,
		Enabled:        resp.ZeroConfiguration.Enabled,
		Addresses:      resp.ZeroConfiguration.Addresses,
	}, nil
}

// SetZeroConfiguration sets the zero-configuration
func (c *Client) SetZeroConfiguration(ctx context.Context, interfaceToken string, enabled bool) error {
	type SetZeroConfiguration struct {
		XMLName        xml.Name `xml:"tds:SetZeroConfiguration"`
		Xmlns          string   `xml:"xmlns:tds,attr"`
		InterfaceToken string   `xml:"tds:InterfaceToken"`
		Enabled        bool     `xml:"tds:Enabled"`
	}

	req := SetZeroConfiguration{
		Xmlns:          deviceNamespace,
		InterfaceToken: interfaceToken,
		Enabled:        enabled,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetZeroConfiguration failed: %w", err)
	}

	return nil
}

// GetDynamicDNS gets the dynamic DNS settings from a device
func (c *Client) GetDynamicDNS(ctx context.Context) (*DynamicDNSInformation, error) {
	type GetDynamicDNS struct {
		XMLName xml.Name `xml:"tds:GetDynamicDNS"`
		Xmlns   string   `xml:"xmlns:tds,attr"`
	}

	type GetDynamicDNSResponse struct {
		XMLName               xml.Name `xml:"GetDynamicDNSResponse"`
		DynamicDNSInformation struct {
			Type string `xml:"Type"`
			Name string `xml:"Name"`
			TTL  string `xml:"TTL"`
		} `xml:"DynamicDNSInformation"`
	}

	req := GetDynamicDNS{
		Xmlns: deviceNamespace,
	}

	var resp GetDynamicDNSResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetDynamicDNS failed: %w", err)
	}

	return &DynamicDNSInformation{
		Type: DynamicDNSType(resp.DynamicDNSInformation.Type),
		Name: resp.DynamicDNSInformation.Name,
		// TTL would need duration parsing
	}, nil
}

// SetDynamicDNS sets the dynamic DNS settings on a device
func (c *Client) SetDynamicDNS(ctx context.Context, dnsType DynamicDNSType, name string) error {
	type SetDynamicDNS struct {
		XMLName xml.Name       `xml:"tds:SetDynamicDNS"`
		Xmlns   string         `xml:"xmlns:tds,attr"`
		Type    DynamicDNSType `xml:"tds:Type"`
		Name    string         `xml:"tds:Name,omitempty"`
	}

	req := SetDynamicDNS{
		Xmlns: deviceNamespace,
		Type:  dnsType,
		Name:  name,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetDynamicDNS failed: %w", err)
	}

	return nil
}

// GetPasswordComplexityConfiguration retrieves the current password complexity configuration settings
func (c *Client) GetPasswordComplexityConfiguration(ctx context.Context) (*PasswordComplexityConfiguration, error) {
	type GetPasswordComplexityConfiguration struct {
		XMLName xml.Name `xml:"tds:GetPasswordComplexityConfiguration"`
		Xmlns   string   `xml:"xmlns:tds,attr"`
	}

	type GetPasswordComplexityConfigurationResponse struct {
		XMLName                   xml.Name `xml:"GetPasswordComplexityConfigurationResponse"`
		MinLen                    int      `xml:"MinLen"`
		Uppercase                 int      `xml:"Uppercase"`
		Number                    int      `xml:"Number"`
		SpecialChars              int      `xml:"SpecialChars"`
		BlockUsernameOccurrence   bool     `xml:"BlockUsernameOccurrence"`
		PolicyConfigurationLocked bool     `xml:"PolicyConfigurationLocked"`
	}

	req := GetPasswordComplexityConfiguration{
		Xmlns: deviceNamespace,
	}

	var resp GetPasswordComplexityConfigurationResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetPasswordComplexityConfiguration failed: %w", err)
	}

	return &PasswordComplexityConfiguration{
		MinLen:                    resp.MinLen,
		Uppercase:                 resp.Uppercase,
		Number:                    resp.Number,
		SpecialChars:              resp.SpecialChars,
		BlockUsernameOccurrence:   resp.BlockUsernameOccurrence,
		PolicyConfigurationLocked: resp.PolicyConfigurationLocked,
	}, nil
}

// SetPasswordComplexityConfiguration allows setting of the password complexity configuration
func (c *Client) SetPasswordComplexityConfiguration(ctx context.Context, config *PasswordComplexityConfiguration) error {
	type SetPasswordComplexityConfiguration struct {
		XMLName                   xml.Name `xml:"tds:SetPasswordComplexityConfiguration"`
		Xmlns                     string   `xml:"xmlns:tds,attr"`
		MinLen                    int      `xml:"tds:MinLen,omitempty"`
		Uppercase                 int      `xml:"tds:Uppercase,omitempty"`
		Number                    int      `xml:"tds:Number,omitempty"`
		SpecialChars              int      `xml:"tds:SpecialChars,omitempty"`
		BlockUsernameOccurrence   bool     `xml:"tds:BlockUsernameOccurrence,omitempty"`
		PolicyConfigurationLocked bool     `xml:"tds:PolicyConfigurationLocked,omitempty"`
	}

	req := SetPasswordComplexityConfiguration{
		Xmlns:                     deviceNamespace,
		MinLen:                    config.MinLen,
		Uppercase:                 config.Uppercase,
		Number:                    config.Number,
		SpecialChars:              config.SpecialChars,
		BlockUsernameOccurrence:   config.BlockUsernameOccurrence,
		PolicyConfigurationLocked: config.PolicyConfigurationLocked,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetPasswordComplexityConfiguration failed: %w", err)
	}

	return nil
}

// GetPasswordHistoryConfiguration retrieves the current password history configuration settings
func (c *Client) GetPasswordHistoryConfiguration(ctx context.Context) (*PasswordHistoryConfiguration, error) {
	type GetPasswordHistoryConfiguration struct {
		XMLName xml.Name `xml:"tds:GetPasswordHistoryConfiguration"`
		Xmlns   string   `xml:"xmlns:tds,attr"`
	}

	type GetPasswordHistoryConfigurationResponse struct {
		XMLName xml.Name `xml:"GetPasswordHistoryConfigurationResponse"`
		Enabled bool     `xml:"Enabled"`
		Length  int      `xml:"Length"`
	}

	req := GetPasswordHistoryConfiguration{
		Xmlns: deviceNamespace,
	}

	var resp GetPasswordHistoryConfigurationResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetPasswordHistoryConfiguration failed: %w", err)
	}

	return &PasswordHistoryConfiguration{
		Enabled: resp.Enabled,
		Length:  resp.Length,
	}, nil
}

// SetPasswordHistoryConfiguration allows setting of the password history configuration
func (c *Client) SetPasswordHistoryConfiguration(ctx context.Context, config *PasswordHistoryConfiguration) error {
	type SetPasswordHistoryConfiguration struct {
		XMLName xml.Name `xml:"tds:SetPasswordHistoryConfiguration"`
		Xmlns   string   `xml:"xmlns:tds,attr"`
		Enabled bool     `xml:"tds:Enabled"`
		Length  int      `xml:"tds:Length"`
	}

	req := SetPasswordHistoryConfiguration{
		Xmlns:   deviceNamespace,
		Enabled: config.Enabled,
		Length:  config.Length,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetPasswordHistoryConfiguration failed: %w", err)
	}

	return nil
}

// GetAuthFailureWarningConfiguration retrieves the current authentication failure warning configuration
func (c *Client) GetAuthFailureWarningConfiguration(ctx context.Context) (*AuthFailureWarningConfiguration, error) {
	type GetAuthFailureWarningConfiguration struct {
		XMLName xml.Name `xml:"tds:GetAuthFailureWarningConfiguration"`
		Xmlns   string   `xml:"xmlns:tds,attr"`
	}

	type GetAuthFailureWarningConfigurationResponse struct {
		XMLName         xml.Name `xml:"GetAuthFailureWarningConfigurationResponse"`
		Enabled         bool     `xml:"Enabled"`
		MonitorPeriod   int      `xml:"MonitorPeriod"`
		MaxAuthFailures int      `xml:"MaxAuthFailures"`
	}

	req := GetAuthFailureWarningConfiguration{
		Xmlns: deviceNamespace,
	}

	var resp GetAuthFailureWarningConfigurationResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetAuthFailureWarningConfiguration failed: %w", err)
	}

	return &AuthFailureWarningConfiguration{
		Enabled:         resp.Enabled,
		MonitorPeriod:   resp.MonitorPeriod,
		MaxAuthFailures: resp.MaxAuthFailures,
	}, nil
}

// SetAuthFailureWarningConfiguration allows setting of the authentication failure warning configuration
func (c *Client) SetAuthFailureWarningConfiguration(ctx context.Context, config *AuthFailureWarningConfiguration) error {
	type SetAuthFailureWarningConfiguration struct {
		XMLName         xml.Name `xml:"tds:SetAuthFailureWarningConfiguration"`
		Xmlns           string   `xml:"xmlns:tds,attr"`
		Enabled         bool     `xml:"tds:Enabled"`
		MonitorPeriod   int      `xml:"tds:MonitorPeriod"`
		MaxAuthFailures int      `xml:"tds:MaxAuthFailures"`
	}

	req := SetAuthFailureWarningConfiguration{
		Xmlns:           deviceNamespace,
		Enabled:         config.Enabled,
		MonitorPeriod:   config.MonitorPeriod,
		MaxAuthFailures: config.MaxAuthFailures,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, c.endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetAuthFailureWarningConfiguration failed: %w", err)
	}

	return nil
}
