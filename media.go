package onvif

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/0x524a/onvif-go/internal/soap"
)

// Media service namespace
const mediaNamespace = "http://www.onvif.org/ver10/media/wsdl"

// GetProfiles retrieves all media profiles
func (c *Client) GetProfiles(ctx context.Context) ([]*Profile, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetProfiles struct {
		XMLName xml.Name `xml:"trt:GetProfiles"`
		Xmlns   string   `xml:"xmlns:trt,attr"`
	}

	type GetProfilesResponse struct {
		XMLName  xml.Name `xml:"GetProfilesResponse"`
		Profiles []struct {
			Token                    string `xml:"token,attr"`
			Name                     string `xml:"Name"`
			VideoSourceConfiguration *struct {
				Token       string `xml:"token,attr"`
				Name        string `xml:"Name"`
				UseCount    int    `xml:"UseCount"`
				SourceToken string `xml:"SourceToken"`
				Bounds      *struct {
					X      int `xml:"x,attr"`
					Y      int `xml:"y,attr"`
					Width  int `xml:"width,attr"`
					Height int `xml:"height,attr"`
				} `xml:"Bounds"`
			} `xml:"VideoSourceConfiguration"`
			VideoEncoderConfiguration *struct {
				Token      string `xml:"token,attr"`
				Name       string `xml:"Name"`
				UseCount   int    `xml:"UseCount"`
				Encoding   string `xml:"Encoding"`
				Resolution *struct {
					Width  int `xml:"Width"`
					Height int `xml:"Height"`
				} `xml:"Resolution"`
				Quality     float64 `xml:"Quality"`
				RateControl *struct {
					FrameRateLimit   int `xml:"FrameRateLimit"`
					EncodingInterval int `xml:"EncodingInterval"`
					BitrateLimit     int `xml:"BitrateLimit"`
				} `xml:"RateControl"`
			} `xml:"VideoEncoderConfiguration"`
			PTZConfiguration *struct {
				Token     string `xml:"token,attr"`
				Name      string `xml:"Name"`
				UseCount  int    `xml:"UseCount"`
				NodeToken string `xml:"NodeToken"`
			} `xml:"PTZConfiguration"`
		} `xml:"Profiles"`
	}

	req := GetProfiles{
		Xmlns: mediaNamespace,
	}

	var resp GetProfilesResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetProfiles failed: %w", err)
	}

	profiles := make([]*Profile, len(resp.Profiles))
	for i, p := range resp.Profiles {
		profile := &Profile{
			Token: p.Token,
			Name:  p.Name,
		}

		if p.VideoSourceConfiguration != nil {
			profile.VideoSourceConfiguration = &VideoSourceConfiguration{
				Token:       p.VideoSourceConfiguration.Token,
				Name:        p.VideoSourceConfiguration.Name,
				UseCount:    p.VideoSourceConfiguration.UseCount,
				SourceToken: p.VideoSourceConfiguration.SourceToken,
			}
			if p.VideoSourceConfiguration.Bounds != nil {
				profile.VideoSourceConfiguration.Bounds = &IntRectangle{
					X:      p.VideoSourceConfiguration.Bounds.X,
					Y:      p.VideoSourceConfiguration.Bounds.Y,
					Width:  p.VideoSourceConfiguration.Bounds.Width,
					Height: p.VideoSourceConfiguration.Bounds.Height,
				}
			}
		}

		if p.VideoEncoderConfiguration != nil {
			profile.VideoEncoderConfiguration = &VideoEncoderConfiguration{
				Token:    p.VideoEncoderConfiguration.Token,
				Name:     p.VideoEncoderConfiguration.Name,
				UseCount: p.VideoEncoderConfiguration.UseCount,
				Encoding: p.VideoEncoderConfiguration.Encoding,
				Quality:  p.VideoEncoderConfiguration.Quality,
			}
			if p.VideoEncoderConfiguration.Resolution != nil {
				profile.VideoEncoderConfiguration.Resolution = &VideoResolution{
					Width:  p.VideoEncoderConfiguration.Resolution.Width,
					Height: p.VideoEncoderConfiguration.Resolution.Height,
				}
			}
			if p.VideoEncoderConfiguration.RateControl != nil {
				profile.VideoEncoderConfiguration.RateControl = &VideoRateControl{
					FrameRateLimit:   p.VideoEncoderConfiguration.RateControl.FrameRateLimit,
					EncodingInterval: p.VideoEncoderConfiguration.RateControl.EncodingInterval,
					BitrateLimit:     p.VideoEncoderConfiguration.RateControl.BitrateLimit,
				}
			}
		}

		if p.PTZConfiguration != nil {
			profile.PTZConfiguration = &PTZConfiguration{
				Token:     p.PTZConfiguration.Token,
				Name:      p.PTZConfiguration.Name,
				UseCount:  p.PTZConfiguration.UseCount,
				NodeToken: p.PTZConfiguration.NodeToken,
			}
		}

		profiles[i] = profile
	}

	return profiles, nil
}

// GetStreamURI retrieves the stream URI for a profile
func (c *Client) GetStreamURI(ctx context.Context, profileToken string) (*MediaURI, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetStreamUri struct {
		XMLName     xml.Name `xml:"trt:GetStreamUri"`
		Xmlns       string   `xml:"xmlns:trt,attr"`
		Xmlnst      string   `xml:"xmlns:tt,attr"`
		StreamSetup struct {
			Stream    string `xml:"tt:Stream"`
			Transport struct {
				Protocol string `xml:"tt:Protocol"`
			} `xml:"tt:Transport"`
		} `xml:"trt:StreamSetup"`
		ProfileToken string `xml:"trt:ProfileToken"`
	}

	type GetStreamUriResponse struct {
		XMLName  xml.Name `xml:"GetStreamUriResponse"`
		MediaUri struct {
			Uri                 string `xml:"Uri"`
			InvalidAfterConnect bool   `xml:"InvalidAfterConnect"`
			InvalidAfterReboot  bool   `xml:"InvalidAfterReboot"`
			Timeout             string `xml:"Timeout"`
		} `xml:"MediaUri"`
	}

	req := GetStreamUri{
		Xmlns:        mediaNamespace,
		Xmlnst:       "http://www.onvif.org/ver10/schema",
		ProfileToken: profileToken,
	}
	req.StreamSetup.Stream = "RTP-Unicast"
	req.StreamSetup.Transport.Protocol = "RTSP"

	var resp GetStreamUriResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetStreamUri failed: %w", err)
	}

	return &MediaURI{
		URI:                 resp.MediaUri.Uri,
		InvalidAfterConnect: resp.MediaUri.InvalidAfterConnect,
		InvalidAfterReboot:  resp.MediaUri.InvalidAfterReboot,
	}, nil
}

// GetSnapshotURI retrieves the snapshot URI for a profile
func (c *Client) GetSnapshotURI(ctx context.Context, profileToken string) (*MediaURI, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetSnapshotUri struct {
		XMLName      xml.Name `xml:"trt:GetSnapshotUri"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	type GetSnapshotUriResponse struct {
		XMLName  xml.Name `xml:"GetSnapshotUriResponse"`
		MediaUri struct {
			Uri                 string `xml:"Uri"`
			InvalidAfterConnect bool   `xml:"InvalidAfterConnect"`
			InvalidAfterReboot  bool   `xml:"InvalidAfterReboot"`
			Timeout             string `xml:"Timeout"`
		} `xml:"MediaUri"`
	}

	req := GetSnapshotUri{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	var resp GetSnapshotUriResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetSnapshotUri failed: %w", err)
	}

	return &MediaURI{
		URI:                 resp.MediaUri.Uri,
		InvalidAfterConnect: resp.MediaUri.InvalidAfterConnect,
		InvalidAfterReboot:  resp.MediaUri.InvalidAfterReboot,
	}, nil
}

// GetVideoEncoderConfiguration retrieves video encoder configuration
func (c *Client) GetVideoEncoderConfiguration(ctx context.Context, configurationToken string) (*VideoEncoderConfiguration, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetVideoEncoderConfiguration struct {
		XMLName            xml.Name `xml:"trt:GetVideoEncoderConfiguration"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	type GetVideoEncoderConfigurationResponse struct {
		XMLName       xml.Name `xml:"GetVideoEncoderConfigurationResponse"`
		Configuration struct {
			Token      string `xml:"token,attr"`
			Name       string `xml:"Name"`
			UseCount   int    `xml:"UseCount"`
			Encoding   string `xml:"Encoding"`
			Resolution *struct {
				Width  int `xml:"Width"`
				Height int `xml:"Height"`
			} `xml:"Resolution"`
			Quality     float64 `xml:"Quality"`
			RateControl *struct {
				FrameRateLimit   int `xml:"FrameRateLimit"`
				EncodingInterval int `xml:"EncodingInterval"`
				BitrateLimit     int `xml:"BitrateLimit"`
			} `xml:"RateControl"`
		} `xml:"Configuration"`
	}

	req := GetVideoEncoderConfiguration{
		Xmlns:              mediaNamespace,
		ConfigurationToken: configurationToken,
	}

	var resp GetVideoEncoderConfigurationResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetVideoEncoderConfiguration failed: %w", err)
	}

	config := &VideoEncoderConfiguration{
		Token:    resp.Configuration.Token,
		Name:     resp.Configuration.Name,
		UseCount: resp.Configuration.UseCount,
		Encoding: resp.Configuration.Encoding,
		Quality:  resp.Configuration.Quality,
	}

	if resp.Configuration.Resolution != nil {
		config.Resolution = &VideoResolution{
			Width:  resp.Configuration.Resolution.Width,
			Height: resp.Configuration.Resolution.Height,
		}
	}

	if resp.Configuration.RateControl != nil {
		config.RateControl = &VideoRateControl{
			FrameRateLimit:   resp.Configuration.RateControl.FrameRateLimit,
			EncodingInterval: resp.Configuration.RateControl.EncodingInterval,
			BitrateLimit:     resp.Configuration.RateControl.BitrateLimit,
		}
	}

	return config, nil
}

// GetVideoSources retrieves all video sources
func (c *Client) GetVideoSources(ctx context.Context) ([]*VideoSource, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetVideoSources struct {
		XMLName xml.Name `xml:"trt:GetVideoSources"`
		Xmlns   string   `xml:"xmlns:trt,attr"`
	}

	type GetVideoSourcesResponse struct {
		XMLName      xml.Name `xml:"GetVideoSourcesResponse"`
		VideoSources []struct {
			Token      string  `xml:"token,attr"`
			Framerate  float64 `xml:"Framerate"`
			Resolution struct {
				Width  int `xml:"Width"`
				Height int `xml:"Height"`
			} `xml:"Resolution"`
		} `xml:"VideoSources"`
	}

	req := GetVideoSources{
		Xmlns: mediaNamespace,
	}

	var resp GetVideoSourcesResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetVideoSources failed: %w", err)
	}

	sources := make([]*VideoSource, len(resp.VideoSources))
	for i, s := range resp.VideoSources {
		sources[i] = &VideoSource{
			Token:     s.Token,
			Framerate: s.Framerate,
			Resolution: &VideoResolution{
				Width:  s.Resolution.Width,
				Height: s.Resolution.Height,
			},
		}
	}

	return sources, nil
}

// GetAudioSources retrieves all audio sources
func (c *Client) GetAudioSources(ctx context.Context) ([]*AudioSource, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetAudioSources struct {
		XMLName xml.Name `xml:"trt:GetAudioSources"`
		Xmlns   string   `xml:"xmlns:trt,attr"`
	}

	type GetAudioSourcesResponse struct {
		XMLName      xml.Name `xml:"GetAudioSourcesResponse"`
		AudioSources []struct {
			Token    string `xml:"token,attr"`
			Channels int    `xml:"Channels"`
		} `xml:"AudioSources"`
	}

	req := GetAudioSources{
		Xmlns: mediaNamespace,
	}

	var resp GetAudioSourcesResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetAudioSources failed: %w", err)
	}

	sources := make([]*AudioSource, len(resp.AudioSources))
	for i, s := range resp.AudioSources {
		sources[i] = &AudioSource{
			Token:    s.Token,
			Channels: s.Channels,
		}
	}

	return sources, nil
}

// GetAudioOutputs retrieves all audio outputs
func (c *Client) GetAudioOutputs(ctx context.Context) ([]*AudioOutput, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetAudioOutputs struct {
		XMLName xml.Name `xml:"trt:GetAudioOutputs"`
		Xmlns   string   `xml:"xmlns:trt,attr"`
	}

	type GetAudioOutputsResponse struct {
		XMLName      xml.Name `xml:"GetAudioOutputsResponse"`
		AudioOutputs []struct {
			Token string `xml:"token,attr"`
		} `xml:"AudioOutputs"`
	}

	req := GetAudioOutputs{
		Xmlns: mediaNamespace,
	}

	var resp GetAudioOutputsResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetAudioOutputs failed: %w", err)
	}

	outputs := make([]*AudioOutput, len(resp.AudioOutputs))
	for i, o := range resp.AudioOutputs {
		outputs[i] = &AudioOutput{
			Token: o.Token,
		}
	}

	return outputs, nil
}

// CreateProfile creates a new media profile
func (c *Client) CreateProfile(ctx context.Context, name, token string) (*Profile, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type CreateProfile struct {
		XMLName xml.Name `xml:"trt:CreateProfile"`
		Xmlns   string   `xml:"xmlns:trt,attr"`
		Name    string   `xml:"trt:Name"`
		Token   *string  `xml:"trt:Token,omitempty"`
	}

	type CreateProfileResponse struct {
		XMLName xml.Name `xml:"CreateProfileResponse"`
		Profile struct {
			Token string `xml:"token,attr"`
			Name  string `xml:"Name"`
		} `xml:"Profile"`
	}

	req := CreateProfile{
		Xmlns: mediaNamespace,
		Name:  name,
	}
	if token != "" {
		req.Token = &token
	}

	var resp CreateProfileResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("CreateProfile failed: %w", err)
	}

	return &Profile{
		Token: resp.Profile.Token,
		Name:  resp.Profile.Name,
	}, nil
}

// DeleteProfile deletes a media profile
func (c *Client) DeleteProfile(ctx context.Context, profileToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type DeleteProfile struct {
		XMLName      xml.Name `xml:"trt:DeleteProfile"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	req := DeleteProfile{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("DeleteProfile failed: %w", err)
	}

	return nil
}

// SetVideoEncoderConfiguration sets video encoder configuration
func (c *Client) SetVideoEncoderConfiguration(ctx context.Context, config *VideoEncoderConfiguration, forcePersistence bool) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type SetVideoEncoderConfiguration struct {
		XMLName       xml.Name `xml:"trt:SetVideoEncoderConfiguration"`
		Xmlns         string   `xml:"xmlns:trt,attr"`
		Xmlnst        string   `xml:"xmlns:tt,attr"`
		Configuration struct {
			Token      string `xml:"token,attr"`
			Name       string `xml:"tt:Name"`
			UseCount   int    `xml:"tt:UseCount"`
			Encoding   string `xml:"tt:Encoding"`
			Resolution *struct {
				Width  int `xml:"tt:Width"`
				Height int `xml:"tt:Height"`
			} `xml:"tt:Resolution,omitempty"`
			Quality     *float64 `xml:"tt:Quality,omitempty"`
			RateControl *struct {
				FrameRateLimit   int `xml:"tt:FrameRateLimit"`
				EncodingInterval int `xml:"tt:EncodingInterval"`
				BitrateLimit     int `xml:"tt:BitrateLimit"`
			} `xml:"tt:RateControl,omitempty"`
		} `xml:"trt:Configuration"`
		ForcePersistence bool `xml:"trt:ForcePersistence"`
	}

	req := SetVideoEncoderConfiguration{
		Xmlns:            mediaNamespace,
		Xmlnst:           "http://www.onvif.org/ver10/schema",
		ForcePersistence: forcePersistence,
	}

	req.Configuration.Token = config.Token
	req.Configuration.Name = config.Name
	req.Configuration.UseCount = config.UseCount
	req.Configuration.Encoding = config.Encoding

	if config.Resolution != nil {
		req.Configuration.Resolution = &struct {
			Width  int `xml:"tt:Width"`
			Height int `xml:"tt:Height"`
		}{
			Width:  config.Resolution.Width,
			Height: config.Resolution.Height,
		}
	}

	if config.Quality > 0 {
		req.Configuration.Quality = &config.Quality
	}

	if config.RateControl != nil {
		req.Configuration.RateControl = &struct {
			FrameRateLimit   int `xml:"tt:FrameRateLimit"`
			EncodingInterval int `xml:"tt:EncodingInterval"`
			BitrateLimit     int `xml:"tt:BitrateLimit"`
		}{
			FrameRateLimit:   config.RateControl.FrameRateLimit,
			EncodingInterval: config.RateControl.EncodingInterval,
			BitrateLimit:     config.RateControl.BitrateLimit,
		}
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetVideoEncoderConfiguration failed: %w", err)
	}

	return nil
}

// GetMediaServiceCapabilities retrieves media service capabilities
func (c *Client) GetMediaServiceCapabilities(ctx context.Context) (*MediaServiceCapabilities, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetServiceCapabilities struct {
		XMLName xml.Name `xml:"trt:GetServiceCapabilities"`
		Xmlns   string   `xml:"xmlns:trt,attr"`
	}

	type GetServiceCapabilitiesResponse struct {
		XMLName      xml.Name `xml:"GetServiceCapabilitiesResponse"`
		Capabilities struct {
			SnapshotUri         bool `xml:"SnapshotUri,attr"`
			Rotation            bool `xml:"Rotation,attr"`
			VideoSourceMode     bool `xml:"VideoSourceMode,attr"`
			OSD                 bool `xml:"OSD,attr"`
			TemporaryOSDText    bool `xml:"TemporaryOSDText,attr"`
			EXICompression      bool `xml:"EXICompression,attr"`
			ProfileCapabilities *struct {
				MaximumNumberOfProfiles int `xml:"MaximumNumberOfProfiles,attr"`
			} `xml:"ProfileCapabilities"`
			StreamingCapabilities *struct {
				RTPMulticast bool `xml:"RTPMulticast,attr"`
				RTP_TCP      bool `xml:"RTP_TCP,attr"`
				RTP_RTSP_TCP bool `xml:"RTP_RTSP_TCP,attr"`
			} `xml:"StreamingCapabilities"`
		} `xml:"Capabilities"`
	}

	req := GetServiceCapabilities{
		Xmlns: mediaNamespace,
	}

	var resp GetServiceCapabilitiesResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetMediaServiceCapabilities failed: %w", err)
	}

	caps := &MediaServiceCapabilities{
		SnapshotUri:      resp.Capabilities.SnapshotUri,
		Rotation:         resp.Capabilities.Rotation,
		VideoSourceMode:  resp.Capabilities.VideoSourceMode,
		OSD:              resp.Capabilities.OSD,
		TemporaryOSDText: resp.Capabilities.TemporaryOSDText,
		EXICompression:   resp.Capabilities.EXICompression,
	}

	if resp.Capabilities.ProfileCapabilities != nil {
		caps.MaximumNumberOfProfiles = resp.Capabilities.ProfileCapabilities.MaximumNumberOfProfiles
	}

	if resp.Capabilities.StreamingCapabilities != nil {
		caps.RTPMulticast = resp.Capabilities.StreamingCapabilities.RTPMulticast
		caps.RTP_TCP = resp.Capabilities.StreamingCapabilities.RTP_TCP
		caps.RTP_RTSP_TCP = resp.Capabilities.StreamingCapabilities.RTP_RTSP_TCP
	}

	return caps, nil
}

// GetVideoEncoderConfigurationOptions retrieves available options for video encoder configuration
func (c *Client) GetVideoEncoderConfigurationOptions(ctx context.Context, configurationToken string) (*VideoEncoderConfigurationOptions, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetVideoEncoderConfigurationOptions struct {
		XMLName            xml.Name `xml:"trt:GetVideoEncoderConfigurationOptions"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken,omitempty"`
		ProfileToken       string   `xml:"trt:ProfileToken,omitempty"`
	}

	type GetVideoEncoderConfigurationOptionsResponse struct {
		XMLName xml.Name `xml:"GetVideoEncoderConfigurationOptionsResponse"`
		Options struct {
			QualityRange *struct {
				Min float64 `xml:"Min"`
				Max float64 `xml:"Max"`
			} `xml:"QualityRange"`
			JPEG *struct {
				ResolutionsAvailable []struct {
					Width  int `xml:"Width"`
					Height int `xml:"Height"`
				} `xml:"ResolutionsAvailable"`
				FrameRateRange *struct {
					Min float64 `xml:"Min"`
					Max float64 `xml:"Max"`
				} `xml:"FrameRateRange"`
				EncodingIntervalRange *struct {
					Min int `xml:"Min"`
					Max int `xml:"Max"`
				} `xml:"EncodingIntervalRange"`
			} `xml:"JPEG"`
			H264 *struct {
				ResolutionsAvailable []struct {
					Width  int `xml:"Width"`
					Height int `xml:"Height"`
				} `xml:"ResolutionsAvailable"`
				GovLengthRange *struct {
					Min int `xml:"Min"`
					Max int `xml:"Max"`
				} `xml:"GovLengthRange"`
				FrameRateRange *struct {
					Min float64 `xml:"Min"`
					Max float64 `xml:"Max"`
				} `xml:"FrameRateRange"`
				EncodingIntervalRange *struct {
					Min int `xml:"Min"`
					Max int `xml:"Max"`
				} `xml:"EncodingIntervalRange"`
				H264ProfilesSupported []string `xml:"H264ProfilesSupported"`
			} `xml:"H264"`
			Extension struct{} `xml:"Extension"`
		} `xml:"Options"`
	}

	req := GetVideoEncoderConfigurationOptions{
		Xmlns: mediaNamespace,
	}
	if configurationToken != "" {
		req.ConfigurationToken = configurationToken
	}

	var resp GetVideoEncoderConfigurationOptionsResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetVideoEncoderConfigurationOptions failed: %w", err)
	}

	options := &VideoEncoderConfigurationOptions{}

	if resp.Options.QualityRange != nil {
		options.QualityRange = &FloatRange{
			Min: resp.Options.QualityRange.Min,
			Max: resp.Options.QualityRange.Max,
		}
	}

	if resp.Options.JPEG != nil {
		jpegOpts := &JPEGOptions{}
		if resp.Options.JPEG.FrameRateRange != nil {
			jpegOpts.FrameRateRange = &FloatRange{
				Min: resp.Options.JPEG.FrameRateRange.Min,
				Max: resp.Options.JPEG.FrameRateRange.Max,
			}
		}
		if resp.Options.JPEG.EncodingIntervalRange != nil {
			jpegOpts.EncodingIntervalRange = &IntRange{
				Min: resp.Options.JPEG.EncodingIntervalRange.Min,
				Max: resp.Options.JPEG.EncodingIntervalRange.Max,
			}
		}
		for _, res := range resp.Options.JPEG.ResolutionsAvailable {
			jpegOpts.ResolutionsAvailable = append(jpegOpts.ResolutionsAvailable, &VideoResolution{
				Width:  res.Width,
				Height: res.Height,
			})
		}
		options.JPEG = jpegOpts
	}

	if resp.Options.H264 != nil {
		h264Opts := &H264Options{}
		if resp.Options.H264.FrameRateRange != nil {
			h264Opts.FrameRateRange = &FloatRange{
				Min: resp.Options.H264.FrameRateRange.Min,
				Max: resp.Options.H264.FrameRateRange.Max,
			}
		}
		if resp.Options.H264.GovLengthRange != nil {
			h264Opts.GovLengthRange = &IntRange{
				Min: resp.Options.H264.GovLengthRange.Min,
				Max: resp.Options.H264.GovLengthRange.Max,
			}
		}
		if resp.Options.H264.EncodingIntervalRange != nil {
			h264Opts.EncodingIntervalRange = &IntRange{
				Min: resp.Options.H264.EncodingIntervalRange.Min,
				Max: resp.Options.H264.EncodingIntervalRange.Max,
			}
		}
		for _, res := range resp.Options.H264.ResolutionsAvailable {
			h264Opts.ResolutionsAvailable = append(h264Opts.ResolutionsAvailable, &VideoResolution{
				Width:  res.Width,
				Height: res.Height,
			})
		}
		h264Opts.H264ProfilesSupported = resp.Options.H264.H264ProfilesSupported
		options.H264 = h264Opts
	}

	return options, nil
}

// GetAudioEncoderConfiguration retrieves audio encoder configuration
func (c *Client) GetAudioEncoderConfiguration(ctx context.Context, configurationToken string) (*AudioEncoderConfiguration, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetAudioEncoderConfiguration struct {
		XMLName            xml.Name `xml:"trt:GetAudioEncoderConfiguration"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	type GetAudioEncoderConfigurationResponse struct {
		XMLName       xml.Name `xml:"GetAudioEncoderConfigurationResponse"`
		Configuration struct {
			Token      string `xml:"token,attr"`
			Name       string `xml:"Name"`
			UseCount   int    `xml:"UseCount"`
			Encoding   string `xml:"Encoding"`
			Bitrate    int    `xml:"Bitrate"`
			SampleRate int    `xml:"SampleRate"`
			Multicast  *struct {
				Address *struct {
					Type        string `xml:"Type"`
					IPv4Address string `xml:"IPv4Address"`
					IPv6Address string `xml:"IPv6Address"`
				} `xml:"Address"`
				Port      int  `xml:"Port"`
				TTL       int  `xml:"TTL"`
				AutoStart bool `xml:"AutoStart"`
			} `xml:"Multicast"`
			SessionTimeout string `xml:"SessionTimeout"`
		} `xml:"Configuration"`
	}

	req := GetAudioEncoderConfiguration{
		Xmlns:              mediaNamespace,
		ConfigurationToken: configurationToken,
	}

	var resp GetAudioEncoderConfigurationResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetAudioEncoderConfiguration failed: %w", err)
	}

	config := &AudioEncoderConfiguration{
		Token:      resp.Configuration.Token,
		Name:       resp.Configuration.Name,
		UseCount:   resp.Configuration.UseCount,
		Encoding:   resp.Configuration.Encoding,
		Bitrate:    resp.Configuration.Bitrate,
		SampleRate: resp.Configuration.SampleRate,
	}

	if resp.Configuration.Multicast != nil {
		config.Multicast = &MulticastConfiguration{
			Port:      resp.Configuration.Multicast.Port,
			TTL:       resp.Configuration.Multicast.TTL,
			AutoStart: resp.Configuration.Multicast.AutoStart,
		}
		if resp.Configuration.Multicast.Address != nil {
			config.Multicast.Address = &IPAddress{
				Type:        resp.Configuration.Multicast.Address.Type,
				IPv4Address: resp.Configuration.Multicast.Address.IPv4Address,
				IPv6Address: resp.Configuration.Multicast.Address.IPv6Address,
			}
		}
	}

	return config, nil
}

// SetAudioEncoderConfiguration sets audio encoder configuration
func (c *Client) SetAudioEncoderConfiguration(ctx context.Context, config *AudioEncoderConfiguration, forcePersistence bool) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type SetAudioEncoderConfiguration struct {
		XMLName       xml.Name `xml:"trt:SetAudioEncoderConfiguration"`
		Xmlns         string   `xml:"xmlns:trt,attr"`
		Xmlnst        string   `xml:"xmlns:tt,attr"`
		Configuration struct {
			Token      string `xml:"token,attr"`
			Name       string `xml:"tt:Name"`
			UseCount   int    `xml:"tt:UseCount"`
			Encoding   string `xml:"tt:Encoding"`
			Bitrate    int    `xml:"tt:Bitrate,omitempty"`
			SampleRate int    `xml:"tt:SampleRate,omitempty"`
			Multicast  *struct {
				Address *struct {
					Type        string `xml:"tt:Type"`
					IPv4Address string `xml:"tt:IPv4Address,omitempty"`
					IPv6Address string `xml:"tt:IPv6Address,omitempty"`
				} `xml:"tt:Address,omitempty"`
				Port      int  `xml:"tt:Port,omitempty"`
				TTL       int  `xml:"tt:TTL,omitempty"`
				AutoStart bool `xml:"tt:AutoStart,omitempty"`
			} `xml:"tt:Multicast,omitempty"`
			SessionTimeout string `xml:"tt:SessionTimeout,omitempty"`
		} `xml:"trt:Configuration"`
		ForcePersistence bool `xml:"trt:ForcePersistence"`
	}

	req := SetAudioEncoderConfiguration{
		Xmlns:            mediaNamespace,
		Xmlnst:           "http://www.onvif.org/ver10/schema",
		ForcePersistence: forcePersistence,
	}

	req.Configuration.Token = config.Token
	req.Configuration.Name = config.Name
	req.Configuration.UseCount = config.UseCount
	req.Configuration.Encoding = config.Encoding
	if config.Bitrate > 0 {
		req.Configuration.Bitrate = config.Bitrate
	}
	if config.SampleRate > 0 {
		req.Configuration.SampleRate = config.SampleRate
	}

	if config.Multicast != nil {
		req.Configuration.Multicast = &struct {
			Address *struct {
				Type        string `xml:"tt:Type"`
				IPv4Address string `xml:"tt:IPv4Address,omitempty"`
				IPv6Address string `xml:"tt:IPv6Address,omitempty"`
			} `xml:"tt:Address,omitempty"`
			Port      int  `xml:"tt:Port,omitempty"`
			TTL       int  `xml:"tt:TTL,omitempty"`
			AutoStart bool `xml:"tt:AutoStart,omitempty"`
		}{
			Port:      config.Multicast.Port,
			TTL:       config.Multicast.TTL,
			AutoStart: config.Multicast.AutoStart,
		}
		if config.Multicast.Address != nil {
			req.Configuration.Multicast.Address = &struct {
				Type        string `xml:"tt:Type"`
				IPv4Address string `xml:"tt:IPv4Address,omitempty"`
				IPv6Address string `xml:"tt:IPv6Address,omitempty"`
			}{
				Type:        config.Multicast.Address.Type,
				IPv4Address: config.Multicast.Address.IPv4Address,
				IPv6Address: config.Multicast.Address.IPv6Address,
			}
		}
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetAudioEncoderConfiguration failed: %w", err)
	}

	return nil
}

// GetMetadataConfiguration retrieves metadata configuration
func (c *Client) GetMetadataConfiguration(ctx context.Context, configurationToken string) (*MetadataConfiguration, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetMetadataConfiguration struct {
		XMLName            xml.Name `xml:"trt:GetMetadataConfiguration"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	type GetMetadataConfigurationResponse struct {
		XMLName       xml.Name `xml:"GetMetadataConfigurationResponse"`
		Configuration struct {
			Token     string `xml:"token,attr"`
			Name      string `xml:"Name"`
			UseCount  int    `xml:"UseCount"`
			PTZStatus *struct {
				Status   bool `xml:"Status"`
				Position bool `xml:"Position"`
			} `xml:"PTZStatus"`
			Events    *struct{} `xml:"Events"`
			Analytics bool      `xml:"Analytics"`
			Multicast *struct {
				Address *struct {
					Type        string `xml:"Type"`
					IPv4Address string `xml:"IPv4Address"`
					IPv6Address string `xml:"IPv6Address"`
				} `xml:"Address"`
				Port      int  `xml:"Port"`
				TTL       int  `xml:"TTL"`
				AutoStart bool `xml:"AutoStart"`
			} `xml:"Multicast"`
			SessionTimeout string `xml:"SessionTimeout"`
		} `xml:"Configuration"`
	}

	req := GetMetadataConfiguration{
		Xmlns:              mediaNamespace,
		ConfigurationToken: configurationToken,
	}

	var resp GetMetadataConfigurationResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetMetadataConfiguration failed: %w", err)
	}

	config := &MetadataConfiguration{
		Token:     resp.Configuration.Token,
		Name:      resp.Configuration.Name,
		UseCount:  resp.Configuration.UseCount,
		Analytics: resp.Configuration.Analytics,
	}

	if resp.Configuration.PTZStatus != nil {
		config.PTZStatus = &PTZFilter{
			Status:   resp.Configuration.PTZStatus.Status,
			Position: resp.Configuration.PTZStatus.Position,
		}
	}

	if resp.Configuration.Events != nil {
		config.Events = &EventSubscription{}
	}

	if resp.Configuration.Multicast != nil {
		config.Multicast = &MulticastConfiguration{
			Port:      resp.Configuration.Multicast.Port,
			TTL:       resp.Configuration.Multicast.TTL,
			AutoStart: resp.Configuration.Multicast.AutoStart,
		}
		if resp.Configuration.Multicast.Address != nil {
			config.Multicast.Address = &IPAddress{
				Type:        resp.Configuration.Multicast.Address.Type,
				IPv4Address: resp.Configuration.Multicast.Address.IPv4Address,
				IPv6Address: resp.Configuration.Multicast.Address.IPv6Address,
			}
		}
	}

	return config, nil
}

// SetMetadataConfiguration sets metadata configuration
func (c *Client) SetMetadataConfiguration(ctx context.Context, config *MetadataConfiguration, forcePersistence bool) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type SetMetadataConfiguration struct {
		XMLName       xml.Name `xml:"trt:SetMetadataConfiguration"`
		Xmlns         string   `xml:"xmlns:trt,attr"`
		Xmlnst        string   `xml:"xmlns:tt,attr"`
		Configuration struct {
			Token     string `xml:"token,attr"`
			Name      string `xml:"tt:Name"`
			UseCount  int    `xml:"tt:UseCount"`
			PTZStatus *struct {
				Status   bool `xml:"tt:Status"`
				Position bool `xml:"tt:Position"`
			} `xml:"tt:PTZStatus,omitempty"`
			Events    *struct{} `xml:"tt:Events,omitempty"`
			Analytics bool      `xml:"tt:Analytics,omitempty"`
			Multicast *struct {
				Address *struct {
					Type        string `xml:"tt:Type"`
					IPv4Address string `xml:"tt:IPv4Address,omitempty"`
					IPv6Address string `xml:"tt:IPv6Address,omitempty"`
				} `xml:"tt:Address,omitempty"`
				Port      int  `xml:"tt:Port,omitempty"`
				TTL       int  `xml:"tt:TTL,omitempty"`
				AutoStart bool `xml:"tt:AutoStart,omitempty"`
			} `xml:"tt:Multicast,omitempty"`
			SessionTimeout string `xml:"tt:SessionTimeout,omitempty"`
		} `xml:"trt:Configuration"`
		ForcePersistence bool `xml:"trt:ForcePersistence"`
	}

	req := SetMetadataConfiguration{
		Xmlns:            mediaNamespace,
		Xmlnst:           "http://www.onvif.org/ver10/schema",
		ForcePersistence: forcePersistence,
	}

	req.Configuration.Token = config.Token
	req.Configuration.Name = config.Name
	req.Configuration.UseCount = config.UseCount
	req.Configuration.Analytics = config.Analytics

	if config.PTZStatus != nil {
		req.Configuration.PTZStatus = &struct {
			Status   bool `xml:"tt:Status"`
			Position bool `xml:"tt:Position"`
		}{
			Status:   config.PTZStatus.Status,
			Position: config.PTZStatus.Position,
		}
	}

	if config.Events != nil {
		req.Configuration.Events = &struct{}{}
	}

	if config.Multicast != nil {
		req.Configuration.Multicast = &struct {
			Address *struct {
				Type        string `xml:"tt:Type"`
				IPv4Address string `xml:"tt:IPv4Address,omitempty"`
				IPv6Address string `xml:"tt:IPv6Address,omitempty"`
			} `xml:"tt:Address,omitempty"`
			Port      int  `xml:"tt:Port,omitempty"`
			TTL       int  `xml:"tt:TTL,omitempty"`
			AutoStart bool `xml:"tt:AutoStart,omitempty"`
		}{
			Port:      config.Multicast.Port,
			TTL:       config.Multicast.TTL,
			AutoStart: config.Multicast.AutoStart,
		}
		if config.Multicast.Address != nil {
			req.Configuration.Multicast.Address = &struct {
				Type        string `xml:"tt:Type"`
				IPv4Address string `xml:"tt:IPv4Address,omitempty"`
				IPv6Address string `xml:"tt:IPv6Address,omitempty"`
			}{
				Type:        config.Multicast.Address.Type,
				IPv4Address: config.Multicast.Address.IPv4Address,
				IPv6Address: config.Multicast.Address.IPv6Address,
			}
		}
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetMetadataConfiguration failed: %w", err)
	}

	return nil
}

// GetVideoSourceModes retrieves available video source modes
func (c *Client) GetVideoSourceModes(ctx context.Context, videoSourceToken string) ([]*VideoSourceMode, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetVideoSourceModes struct {
		XMLName          xml.Name `xml:"trt:GetVideoSourceModes"`
		Xmlns            string   `xml:"xmlns:trt,attr"`
		VideoSourceToken string   `xml:"trt:VideoSourceToken"`
	}

	type GetVideoSourceModesResponse struct {
		XMLName          xml.Name `xml:"GetVideoSourceModesResponse"`
		VideoSourceModes []struct {
			Token      string `xml:"token,attr"`
			Enabled    bool   `xml:"Enabled"`
			Resolution struct {
				Width  int `xml:"Width"`
				Height int `xml:"Height"`
			} `xml:"Resolution"`
		} `xml:"VideoSourceModes"`
	}

	req := GetVideoSourceModes{
		Xmlns:            mediaNamespace,
		VideoSourceToken: videoSourceToken,
	}

	var resp GetVideoSourceModesResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetVideoSourceModes failed: %w", err)
	}

	modes := make([]*VideoSourceMode, len(resp.VideoSourceModes))
	for i, m := range resp.VideoSourceModes {
		modes[i] = &VideoSourceMode{
			Token:   m.Token,
			Enabled: m.Enabled,
			Resolution: &VideoResolution{
				Width:  m.Resolution.Width,
				Height: m.Resolution.Height,
			},
		}
	}

	return modes, nil
}

// SetVideoSourceMode sets the video source mode
func (c *Client) SetVideoSourceMode(ctx context.Context, videoSourceToken, modeToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type SetVideoSourceMode struct {
		XMLName          xml.Name `xml:"trt:SetVideoSourceMode"`
		Xmlns            string   `xml:"xmlns:trt,attr"`
		VideoSourceToken string   `xml:"trt:VideoSourceToken"`
		ModeToken        string   `xml:"trt:ModeToken"`
	}

	req := SetVideoSourceMode{
		Xmlns:            mediaNamespace,
		VideoSourceToken: videoSourceToken,
		ModeToken:        modeToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetVideoSourceMode failed: %w", err)
	}

	return nil
}

// SetSynchronizationPoint sets a synchronization point for the stream
func (c *Client) SetSynchronizationPoint(ctx context.Context, profileToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type SetSynchronizationPoint struct {
		XMLName      xml.Name `xml:"trt:SetSynchronizationPoint"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	req := SetSynchronizationPoint{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetSynchronizationPoint failed: %w", err)
	}

	return nil
}

// GetOSDs retrieves all OSD configurations
func (c *Client) GetOSDs(ctx context.Context, configurationToken string) ([]*OSDConfiguration, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetOSDs struct {
		XMLName            xml.Name `xml:"trt:GetOSDs"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken,omitempty"`
	}

	type GetOSDsResponse struct {
		XMLName xml.Name `xml:"GetOSDsResponse"`
		OSDs    []struct {
			Token string `xml:"token,attr"`
		} `xml:"OSDs"`
	}

	req := GetOSDs{
		Xmlns: mediaNamespace,
	}
	if configurationToken != "" {
		req.ConfigurationToken = configurationToken
	}

	var resp GetOSDsResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetOSDs failed: %w", err)
	}

	osds := make([]*OSDConfiguration, len(resp.OSDs))
	for i, o := range resp.OSDs {
		osds[i] = &OSDConfiguration{
			Token: o.Token,
		}
	}

	return osds, nil
}

// GetOSD retrieves a specific OSD configuration
func (c *Client) GetOSD(ctx context.Context, osdToken string) (*OSDConfiguration, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetOSD struct {
		XMLName  xml.Name `xml:"trt:GetOSD"`
		Xmlns    string   `xml:"xmlns:trt,attr"`
		OSDToken string   `xml:"trt:OSDToken"`
	}

	type GetOSDResponse struct {
		XMLName xml.Name `xml:"GetOSDResponse"`
		OSD     struct {
			Token string `xml:"token,attr"`
		} `xml:"OSD"`
	}

	req := GetOSD{
		Xmlns:    mediaNamespace,
		OSDToken: osdToken,
	}

	var resp GetOSDResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetOSD failed: %w", err)
	}

	return &OSDConfiguration{
		Token: resp.OSD.Token,
	}, nil
}

// SetOSD sets OSD configuration
func (c *Client) SetOSD(ctx context.Context, osd *OSDConfiguration) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type SetOSD struct {
		XMLName xml.Name `xml:"trt:SetOSD"`
		Xmlns   string   `xml:"xmlns:trt,attr"`
		Xmlnst  string   `xml:"xmlns:tt,attr"`
		OSD     struct {
			Token string `xml:"token,attr"`
		} `xml:"trt:OSD"`
	}

	req := SetOSD{
		Xmlns:  mediaNamespace,
		Xmlnst: "http://www.onvif.org/ver10/schema",
	}
	req.OSD.Token = osd.Token

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetOSD failed: %w", err)
	}

	return nil
}

// CreateOSD creates a new OSD configuration
func (c *Client) CreateOSD(ctx context.Context, videoSourceConfigurationToken string, osd *OSDConfiguration) (*OSDConfiguration, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type CreateOSD struct {
		XMLName                       xml.Name `xml:"trt:CreateOSD"`
		Xmlns                         string   `xml:"xmlns:trt,attr"`
		Xmlnst                        string   `xml:"xmlns:tt,attr"`
		VideoSourceConfigurationToken string   `xml:"trt:VideoSourceConfigurationToken"`
		OSD                           struct {
			Token string `xml:"token,attr,omitempty"`
		} `xml:"trt:OSD"`
	}

	type CreateOSDResponse struct {
		XMLName xml.Name `xml:"CreateOSDResponse"`
		OSD     struct {
			Token string `xml:"token,attr"`
		} `xml:"OSD"`
	}

	req := CreateOSD{
		Xmlns:                         mediaNamespace,
		Xmlnst:                        "http://www.onvif.org/ver10/schema",
		VideoSourceConfigurationToken: videoSourceConfigurationToken,
	}
	if osd != nil && osd.Token != "" {
		req.OSD.Token = osd.Token
	}

	var resp CreateOSDResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("CreateOSD failed: %w", err)
	}

	return &OSDConfiguration{
		Token: resp.OSD.Token,
	}, nil
}

// DeleteOSD deletes an OSD configuration
func (c *Client) DeleteOSD(ctx context.Context, osdToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type DeleteOSD struct {
		XMLName  xml.Name `xml:"trt:DeleteOSD"`
		Xmlns    string   `xml:"xmlns:trt,attr"`
		OSDToken string   `xml:"trt:OSDToken"`
	}

	req := DeleteOSD{
		Xmlns:    mediaNamespace,
		OSDToken: osdToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("DeleteOSD failed: %w", err)
	}

	return nil
}

// StartMulticastStreaming starts multicast streaming
func (c *Client) StartMulticastStreaming(ctx context.Context, profileToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type StartMulticastStreaming struct {
		XMLName      xml.Name `xml:"trt:StartMulticastStreaming"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	req := StartMulticastStreaming{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("StartMulticastStreaming failed: %w", err)
	}

	return nil
}

// StopMulticastStreaming stops multicast streaming
func (c *Client) StopMulticastStreaming(ctx context.Context, profileToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type StopMulticastStreaming struct {
		XMLName      xml.Name `xml:"trt:StopMulticastStreaming"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	req := StopMulticastStreaming{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("StopMulticastStreaming failed: %w", err)
	}

	return nil
}

// GetProfile retrieves a specific media profile
func (c *Client) GetProfile(ctx context.Context, profileToken string) (*Profile, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetProfile struct {
		XMLName      xml.Name `xml:"trt:GetProfile"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	type GetProfileResponse struct {
		XMLName xml.Name `xml:"GetProfileResponse"`
		Profile struct {
			Token string `xml:"token,attr"`
			Name  string `xml:"Name"`
		} `xml:"Profile"`
	}

	req := GetProfile{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	var resp GetProfileResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetProfile failed: %w", err)
	}

	return &Profile{
		Token: resp.Profile.Token,
		Name:  resp.Profile.Name,
	}, nil
}

// SetProfile sets profile configuration
func (c *Client) SetProfile(ctx context.Context, profile *Profile) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type SetProfile struct {
		XMLName xml.Name `xml:"trt:SetProfile"`
		Xmlns   string   `xml:"xmlns:trt,attr"`
		Xmlnst  string   `xml:"xmlns:tt,attr"`
		Profile struct {
			Token string `xml:"token,attr"`
			Name  string `xml:"tt:Name"`
		} `xml:"trt:Profile"`
	}

	req := SetProfile{
		Xmlns:  mediaNamespace,
		Xmlnst: "http://www.onvif.org/ver10/schema",
	}
	req.Profile.Token = profile.Token
	req.Profile.Name = profile.Name

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetProfile failed: %w", err)
	}

	return nil
}

// AddVideoEncoderConfiguration adds video encoder configuration to a profile
func (c *Client) AddVideoEncoderConfiguration(ctx context.Context, profileToken, configurationToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type AddVideoEncoderConfiguration struct {
		XMLName            xml.Name `xml:"trt:AddVideoEncoderConfiguration"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ProfileToken       string   `xml:"trt:ProfileToken"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	req := AddVideoEncoderConfiguration{
		Xmlns:              mediaNamespace,
		ProfileToken:       profileToken,
		ConfigurationToken: configurationToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("AddVideoEncoderConfiguration failed: %w", err)
	}

	return nil
}

// RemoveVideoEncoderConfiguration removes video encoder configuration from a profile
func (c *Client) RemoveVideoEncoderConfiguration(ctx context.Context, profileToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type RemoveVideoEncoderConfiguration struct {
		XMLName      xml.Name `xml:"trt:RemoveVideoEncoderConfiguration"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	req := RemoveVideoEncoderConfiguration{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("RemoveVideoEncoderConfiguration failed: %w", err)
	}

	return nil
}

// AddAudioEncoderConfiguration adds audio encoder configuration to a profile
func (c *Client) AddAudioEncoderConfiguration(ctx context.Context, profileToken, configurationToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type AddAudioEncoderConfiguration struct {
		XMLName            xml.Name `xml:"trt:AddAudioEncoderConfiguration"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ProfileToken       string   `xml:"trt:ProfileToken"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	req := AddAudioEncoderConfiguration{
		Xmlns:              mediaNamespace,
		ProfileToken:       profileToken,
		ConfigurationToken: configurationToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("AddAudioEncoderConfiguration failed: %w", err)
	}

	return nil
}

// RemoveAudioEncoderConfiguration removes audio encoder configuration from a profile
func (c *Client) RemoveAudioEncoderConfiguration(ctx context.Context, profileToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type RemoveAudioEncoderConfiguration struct {
		XMLName      xml.Name `xml:"trt:RemoveAudioEncoderConfiguration"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	req := RemoveAudioEncoderConfiguration{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("RemoveAudioEncoderConfiguration failed: %w", err)
	}

	return nil
}

// AddAudioSourceConfiguration adds audio source configuration to a profile
func (c *Client) AddAudioSourceConfiguration(ctx context.Context, profileToken, configurationToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type AddAudioSourceConfiguration struct {
		XMLName            xml.Name `xml:"trt:AddAudioSourceConfiguration"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ProfileToken       string   `xml:"trt:ProfileToken"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	req := AddAudioSourceConfiguration{
		Xmlns:              mediaNamespace,
		ProfileToken:       profileToken,
		ConfigurationToken: configurationToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("AddAudioSourceConfiguration failed: %w", err)
	}

	return nil
}

// RemoveAudioSourceConfiguration removes audio source configuration from a profile
func (c *Client) RemoveAudioSourceConfiguration(ctx context.Context, profileToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type RemoveAudioSourceConfiguration struct {
		XMLName      xml.Name `xml:"trt:RemoveAudioSourceConfiguration"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	req := RemoveAudioSourceConfiguration{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("RemoveAudioSourceConfiguration failed: %w", err)
	}

	return nil
}

// AddVideoSourceConfiguration adds video source configuration to a profile
func (c *Client) AddVideoSourceConfiguration(ctx context.Context, profileToken, configurationToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type AddVideoSourceConfiguration struct {
		XMLName            xml.Name `xml:"trt:AddVideoSourceConfiguration"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ProfileToken       string   `xml:"trt:ProfileToken"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	req := AddVideoSourceConfiguration{
		Xmlns:              mediaNamespace,
		ProfileToken:       profileToken,
		ConfigurationToken: configurationToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("AddVideoSourceConfiguration failed: %w", err)
	}

	return nil
}

// RemoveVideoSourceConfiguration removes video source configuration from a profile
func (c *Client) RemoveVideoSourceConfiguration(ctx context.Context, profileToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type RemoveVideoSourceConfiguration struct {
		XMLName      xml.Name `xml:"trt:RemoveVideoSourceConfiguration"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	req := RemoveVideoSourceConfiguration{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("RemoveVideoSourceConfiguration failed: %w", err)
	}

	return nil
}

// AddPTZConfiguration adds PTZ configuration to a profile
func (c *Client) AddPTZConfiguration(ctx context.Context, profileToken, configurationToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type AddPTZConfiguration struct {
		XMLName            xml.Name `xml:"trt:AddPTZConfiguration"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ProfileToken       string   `xml:"trt:ProfileToken"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	req := AddPTZConfiguration{
		Xmlns:              mediaNamespace,
		ProfileToken:       profileToken,
		ConfigurationToken: configurationToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("AddPTZConfiguration failed: %w", err)
	}

	return nil
}

// RemovePTZConfiguration removes PTZ configuration from a profile
func (c *Client) RemovePTZConfiguration(ctx context.Context, profileToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type RemovePTZConfiguration struct {
		XMLName      xml.Name `xml:"trt:RemovePTZConfiguration"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	req := RemovePTZConfiguration{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("RemovePTZConfiguration failed: %w", err)
	}

	return nil
}

// AddMetadataConfiguration adds metadata configuration to a profile
func (c *Client) AddMetadataConfiguration(ctx context.Context, profileToken, configurationToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type AddMetadataConfiguration struct {
		XMLName            xml.Name `xml:"trt:AddMetadataConfiguration"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ProfileToken       string   `xml:"trt:ProfileToken"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	req := AddMetadataConfiguration{
		Xmlns:              mediaNamespace,
		ProfileToken:       profileToken,
		ConfigurationToken: configurationToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("AddMetadataConfiguration failed: %w", err)
	}

	return nil
}

// RemoveMetadataConfiguration removes metadata configuration from a profile
func (c *Client) RemoveMetadataConfiguration(ctx context.Context, profileToken string) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type RemoveMetadataConfiguration struct {
		XMLName      xml.Name `xml:"trt:RemoveMetadataConfiguration"`
		Xmlns        string   `xml:"xmlns:trt,attr"`
		ProfileToken string   `xml:"trt:ProfileToken"`
	}

	req := RemoveMetadataConfiguration{
		Xmlns:        mediaNamespace,
		ProfileToken: profileToken,
	}

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("RemoveMetadataConfiguration failed: %w", err)
	}

	return nil
}

// GetAudioEncoderConfigurationOptions retrieves available options for audio encoder configuration
func (c *Client) GetAudioEncoderConfigurationOptions(ctx context.Context, configurationToken, profileToken string) (*AudioEncoderConfigurationOptions, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetAudioEncoderConfigurationOptions struct {
		XMLName            xml.Name `xml:"trt:GetAudioEncoderConfigurationOptions"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken,omitempty"`
		ProfileToken       string   `xml:"trt:ProfileToken,omitempty"`
	}

	type GetAudioEncoderConfigurationOptionsResponse struct {
		XMLName xml.Name `xml:"GetAudioEncoderConfigurationOptionsResponse"`
		Options struct {
			EncodingOptions []string `xml:"EncodingOptions"`
			BitrateList     []int    `xml:"BitrateList"`
			SampleRateList  []int    `xml:"SampleRateList"`
		} `xml:"Options"`
	}

	req := GetAudioEncoderConfigurationOptions{
		Xmlns: mediaNamespace,
	}
	if configurationToken != "" {
		req.ConfigurationToken = configurationToken
	}
	if profileToken != "" {
		req.ProfileToken = profileToken
	}

	var resp GetAudioEncoderConfigurationOptionsResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetAudioEncoderConfigurationOptions failed: %w", err)
	}

	return &AudioEncoderConfigurationOptions{
		EncodingOptions: resp.Options.EncodingOptions,
		BitrateList:     resp.Options.BitrateList,
		SampleRateList:  resp.Options.SampleRateList,
	}, nil
}

// GetMetadataConfigurationOptions retrieves available options for metadata configuration
func (c *Client) GetMetadataConfigurationOptions(ctx context.Context, configurationToken, profileToken string) (*MetadataConfigurationOptions, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetMetadataConfigurationOptions struct {
		XMLName            xml.Name `xml:"trt:GetMetadataConfigurationOptions"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken,omitempty"`
		ProfileToken       string   `xml:"trt:ProfileToken,omitempty"`
	}

	type GetMetadataConfigurationOptionsResponse struct {
		XMLName xml.Name `xml:"GetMetadataConfigurationOptionsResponse"`
		Options struct {
			PTZStatusFilterOptions *struct {
				Status   bool `xml:"Status"`
				Position bool `xml:"Position"`
			} `xml:"PTZStatusFilterOptions"`
			Extension struct{} `xml:"Extension"`
		} `xml:"Options"`
	}

	req := GetMetadataConfigurationOptions{
		Xmlns: mediaNamespace,
	}
	if configurationToken != "" {
		req.ConfigurationToken = configurationToken
	}
	if profileToken != "" {
		req.ProfileToken = profileToken
	}

	var resp GetMetadataConfigurationOptionsResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetMetadataConfigurationOptions failed: %w", err)
	}

	options := &MetadataConfigurationOptions{}
	if resp.Options.PTZStatusFilterOptions != nil {
		options.PTZStatusFilterOptions = &PTZFilter{
			Status:   resp.Options.PTZStatusFilterOptions.Status,
			Position: resp.Options.PTZStatusFilterOptions.Position,
		}
	}

	return options, nil
}

// GetAudioOutputConfiguration retrieves audio output configuration
func (c *Client) GetAudioOutputConfiguration(ctx context.Context, configurationToken string) (*AudioOutputConfiguration, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetAudioOutputConfiguration struct {
		XMLName            xml.Name `xml:"trt:GetAudioOutputConfiguration"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	type GetAudioOutputConfigurationResponse struct {
		XMLName       xml.Name `xml:"GetAudioOutputConfigurationResponse"`
		Configuration struct {
			Token       string `xml:"token,attr"`
			Name        string `xml:"Name"`
			UseCount    int    `xml:"UseCount"`
			OutputToken string `xml:"OutputToken"`
		} `xml:"Configuration"`
	}

	req := GetAudioOutputConfiguration{
		Xmlns:              mediaNamespace,
		ConfigurationToken: configurationToken,
	}

	var resp GetAudioOutputConfigurationResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetAudioOutputConfiguration failed: %w", err)
	}

	return &AudioOutputConfiguration{
		Token:       resp.Configuration.Token,
		Name:        resp.Configuration.Name,
		UseCount:    resp.Configuration.UseCount,
		OutputToken: resp.Configuration.OutputToken,
	}, nil
}

// SetAudioOutputConfiguration sets audio output configuration
func (c *Client) SetAudioOutputConfiguration(ctx context.Context, config *AudioOutputConfiguration, forcePersistence bool) error {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type SetAudioOutputConfiguration struct {
		XMLName       xml.Name `xml:"trt:SetAudioOutputConfiguration"`
		Xmlns         string   `xml:"xmlns:trt,attr"`
		Xmlnst        string   `xml:"xmlns:tt,attr"`
		Configuration struct {
			Token       string `xml:"token,attr"`
			Name        string `xml:"tt:Name"`
			UseCount    int    `xml:"tt:UseCount"`
			OutputToken string `xml:"tt:OutputToken"`
		} `xml:"trt:Configuration"`
		ForcePersistence bool `xml:"trt:ForcePersistence"`
	}

	req := SetAudioOutputConfiguration{
		Xmlns:            mediaNamespace,
		Xmlnst:           "http://www.onvif.org/ver10/schema",
		ForcePersistence: forcePersistence,
	}

	req.Configuration.Token = config.Token
	req.Configuration.Name = config.Name
	req.Configuration.UseCount = config.UseCount
	req.Configuration.OutputToken = config.OutputToken

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, nil); err != nil {
		return fmt.Errorf("SetAudioOutputConfiguration failed: %w", err)
	}

	return nil
}

// GetAudioOutputConfigurationOptions retrieves available options for audio output configuration
func (c *Client) GetAudioOutputConfigurationOptions(ctx context.Context, configurationToken string) (*AudioOutputConfigurationOptions, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetAudioOutputConfigurationOptions struct {
		XMLName            xml.Name `xml:"trt:GetAudioOutputConfigurationOptions"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken,omitempty"`
	}

	type GetAudioOutputConfigurationOptionsResponse struct {
		XMLName xml.Name `xml:"GetAudioOutputConfigurationOptionsResponse"`
		Options struct {
			OutputTokensAvailable []string `xml:"OutputTokensAvailable"`
		} `xml:"Options"`
	}

	req := GetAudioOutputConfigurationOptions{
		Xmlns: mediaNamespace,
	}
	if configurationToken != "" {
		req.ConfigurationToken = configurationToken
	}

	var resp GetAudioOutputConfigurationOptionsResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetAudioOutputConfigurationOptions failed: %w", err)
	}

	return &AudioOutputConfigurationOptions{
		OutputTokensAvailable: resp.Options.OutputTokensAvailable,
	}, nil
}

// GetAudioDecoderConfigurationOptions retrieves available options for audio decoder configuration
func (c *Client) GetAudioDecoderConfigurationOptions(ctx context.Context, configurationToken string) (*AudioDecoderConfigurationOptions, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetAudioDecoderConfigurationOptions struct {
		XMLName            xml.Name `xml:"trt:GetAudioDecoderConfigurationOptions"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken,omitempty"`
	}

	type GetAudioDecoderConfigurationOptionsResponse struct {
		XMLName xml.Name `xml:"GetAudioDecoderConfigurationOptionsResponse"`
		Options struct {
			AACDecOptions *struct {
				BitrateList    []int `xml:"BitrateList"`
				SampleRateList []int `xml:"SampleRateList"`
			} `xml:"AACDecOptions"`
			G711DecOptions *struct {
				BitrateList []int `xml:"BitrateList"`
			} `xml:"G711DecOptions"`
			G726DecOptions *struct {
				BitrateList []int `xml:"BitrateList"`
			} `xml:"G726DecOptions"`
		} `xml:"Options"`
	}

	req := GetAudioDecoderConfigurationOptions{
		Xmlns: mediaNamespace,
	}
	if configurationToken != "" {
		req.ConfigurationToken = configurationToken
	}

	var resp GetAudioDecoderConfigurationOptionsResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetAudioDecoderConfigurationOptions failed: %w", err)
	}

	options := &AudioDecoderConfigurationOptions{}
	if resp.Options.AACDecOptions != nil {
		options.AACDecOptions = &AudioDecoderOptions{
			BitrateList:    resp.Options.AACDecOptions.BitrateList,
			SampleRateList: resp.Options.AACDecOptions.SampleRateList,
		}
	}
	if resp.Options.G711DecOptions != nil {
		options.G711DecOptions = &AudioDecoderOptions{
			BitrateList: resp.Options.G711DecOptions.BitrateList,
		}
	}
	if resp.Options.G726DecOptions != nil {
		options.G726DecOptions = &AudioDecoderOptions{
			BitrateList: resp.Options.G726DecOptions.BitrateList,
		}
	}

	return options, nil
}

// GetGuaranteedNumberOfVideoEncoderInstances retrieves the guaranteed number of video encoder instances
func (c *Client) GetGuaranteedNumberOfVideoEncoderInstances(ctx context.Context, configurationToken string) (*GuaranteedNumberOfVideoEncoderInstances, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetGuaranteedNumberOfVideoEncoderInstances struct {
		XMLName            xml.Name `xml:"trt:GetGuaranteedNumberOfVideoEncoderInstances"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken"`
	}

	type GetGuaranteedNumberOfVideoEncoderInstancesResponse struct {
		XMLName     xml.Name `xml:"GetGuaranteedNumberOfVideoEncoderInstancesResponse"`
		TotalNumber int      `xml:"TotalNumber"`
		JPEG        int      `xml:"JPEG"`
		H264        int      `xml:"H264"`
		MPEG4       int      `xml:"MPEG4"`
	}

	req := GetGuaranteedNumberOfVideoEncoderInstances{
		Xmlns:              mediaNamespace,
		ConfigurationToken: configurationToken,
	}

	var resp GetGuaranteedNumberOfVideoEncoderInstancesResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetGuaranteedNumberOfVideoEncoderInstances failed: %w", err)
	}

	return &GuaranteedNumberOfVideoEncoderInstances{
		TotalNumber: resp.TotalNumber,
		JPEG:        resp.JPEG,
		H264:        resp.H264,
		MPEG4:       resp.MPEG4,
	}, nil
}

// GetOSDOptions retrieves available options for OSD configuration
func (c *Client) GetOSDOptions(ctx context.Context, configurationToken string) (*OSDConfigurationOptions, error) {
	endpoint := c.mediaEndpoint
	if endpoint == "" {
		endpoint = c.endpoint
	}

	type GetOSDOptions struct {
		XMLName            xml.Name `xml:"trt:GetOSDOptions"`
		Xmlns              string   `xml:"xmlns:trt,attr"`
		ConfigurationToken string   `xml:"trt:ConfigurationToken,omitempty"`
	}

	type GetOSDOptionsResponse struct {
		XMLName xml.Name `xml:"GetOSDOptionsResponse"`
		Options struct {
			MaximumNumberOfOSDs int `xml:"MaximumNumberOfOSDs"`
		} `xml:"Options"`
	}

	req := GetOSDOptions{
		Xmlns: mediaNamespace,
	}
	if configurationToken != "" {
		req.ConfigurationToken = configurationToken
	}

	var resp GetOSDOptionsResponse

	username, password := c.GetCredentials()
	soapClient := soap.NewClient(c.httpClient, username, password)

	if err := soapClient.Call(ctx, endpoint, "", req, &resp); err != nil {
		return nil, fmt.Errorf("GetOSDOptions failed: %w", err)
	}

	return &OSDConfigurationOptions{
		MaximumNumberOfOSDs: resp.Options.MaximumNumberOfOSDs,
	}, nil
}
