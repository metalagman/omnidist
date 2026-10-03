package config

import "gopkg.in/yaml.v3"

type metadataPresence struct {
	description bool
	keywords    bool
	license     bool
}

func metadataPresenceFor(description string, keywords []string, license string, set metadataPresence) metadataPresence {
	return metadataPresence{
		description: set.description || description != "",
		keywords:    set.keywords || keywords != nil,
		license:     set.license || license != "",
	}
}

func (cfg *Config) inheritMetadata(description *string, keywords *[]string, license *string, set *metadataPresence) {
	*set = metadataPresenceFor(*description, *keywords, *license, *set)
	project := metadataPresenceFor(cfg.Description, cfg.Keywords, cfg.License, cfg.metadataSet)
	if !set.description {
		*description = cfg.Description
		set.description = project.description
	}
	if !set.keywords {
		*keywords = cfg.Keywords
		set.keywords = project.keywords
	}
	*keywords = append([]string(nil), (*keywords)...)
	if !set.license {
		*license = cfg.License
		set.license = project.license
	}
}

func markRawMetadataPresence(raw *rawConfig, scope map[string]interface{}) {
	raw.metadataSet = metadataPresenceIn(scope)
	distributions, _ := scope["distributions"].(map[string]interface{})
	if raw.Distributions.NPM != nil {
		fields, _ := distributions["npm"].(map[string]interface{})
		raw.Distributions.NPM.metadataSet = metadataPresenceIn(fields)
	}
	if raw.Distributions.UV != nil {
		fields, _ := distributions["uv"].(map[string]interface{})
		raw.Distributions.UV.metadataSet = metadataPresenceIn(fields)
	}
	if raw.Distributions.Gem != nil {
		fields, _ := distributions["gem"].(map[string]interface{})
		raw.Distributions.Gem.metadataSet = metadataPresenceIn(fields)
	}
}

func metadataPresenceIn(fields map[string]interface{}) metadataPresence {
	return metadataPresence{
		description: fields["description"] != nil,
		keywords:    fields["keywords"] != nil,
		license:     fields["license"] != nil,
	}
}

func optionalMetadataString(value string, set bool) *string {
	if !set && value == "" {
		return nil
	}
	return &value
}

func optionalMetadataKeywords(value []string, set bool) *[]string {
	if !set && value == nil {
		return nil
	}
	cloned := append([]string{}, value...)
	return &cloned
}

func marshalDistributionMetadata(value interface{}, description string, keywords []string, license string, set metadataPresence) (interface{}, error) {
	set = metadataPresenceFor(description, keywords, license, set)
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return nil, err
	}
	fields := []struct {
		name  string
		value interface{}
		set   bool
	}{
		{"description", description, set.description && description == ""},
		{"keywords", []string{}, set.keywords && len(keywords) == 0},
		{"license", license, set.license && license == ""},
	}
	for _, field := range fields {
		if !field.set {
			continue
		}
		var key, valueNode yaml.Node
		if err := key.Encode(field.name); err != nil {
			return nil, err
		}
		if err := valueNode.Encode(field.value); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, &key, &valueNode)
	}
	return &node, nil
}

// MarshalYAML preserves explicit empty metadata overrides.
func (d NPMDistributionConfig) MarshalYAML() (interface{}, error) {
	type plain NPMDistributionConfig
	return marshalDistributionMetadata(plain(d), d.Description, d.Keywords, d.License, d.metadataSet)
}

// MarshalYAML preserves explicit empty metadata overrides.
func (d UVDistributionConfig) MarshalYAML() (interface{}, error) {
	type plain UVDistributionConfig
	return marshalDistributionMetadata(plain(d), d.Description, d.Keywords, d.License, d.metadataSet)
}

// MarshalYAML preserves explicit empty metadata overrides.
func (d GemDistributionConfig) MarshalYAML() (interface{}, error) {
	type plain GemDistributionConfig
	return marshalDistributionMetadata(plain(d), d.Description, d.Keywords, d.License, d.metadataSet)
}

// DescriptionConfigured reports whether a resolved description was supplied or inherited.
func (d NPMDistributionConfig) DescriptionConfigured() bool {
	return d.metadataSet.description || d.Description != ""
}

// KeywordsConfigured reports whether resolved keywords were supplied, cleared or inherited.
func (d NPMDistributionConfig) KeywordsConfigured() bool {
	return d.metadataSet.keywords || d.Keywords != nil
}

// LicenseConfigured reports whether a resolved license was supplied or inherited.
func (d NPMDistributionConfig) LicenseConfigured() bool {
	return d.metadataSet.license || d.License != ""
}

// DescriptionConfigured reports whether a resolved description was supplied or inherited.
func (d UVDistributionConfig) DescriptionConfigured() bool {
	return d.metadataSet.description || d.Description != ""
}

// KeywordsConfigured reports whether resolved keywords were supplied, cleared or inherited.
func (d UVDistributionConfig) KeywordsConfigured() bool {
	return d.metadataSet.keywords || d.Keywords != nil
}

// LicenseConfigured reports whether a resolved license was supplied or inherited.
func (d UVDistributionConfig) LicenseConfigured() bool {
	return d.metadataSet.license || d.License != ""
}

// DescriptionConfigured reports whether a resolved description was supplied or inherited.
func (d GemDistributionConfig) DescriptionConfigured() bool {
	return d.metadataSet.description || d.Description != ""
}

// KeywordsConfigured reports whether resolved keywords were supplied, cleared or inherited.
func (d GemDistributionConfig) KeywordsConfigured() bool {
	return d.metadataSet.keywords || d.Keywords != nil
}

// LicenseConfigured reports whether a resolved license was supplied or inherited.
func (d GemDistributionConfig) LicenseConfigured() bool {
	return d.metadataSet.license || d.License != ""
}
