package config

// DefaultsConfig is used to configure the defaults used for all templates
type DefaultsConfig struct {
	// LeftDelim is the left delimiter for templating
	LeftDelim *string `mapstructure:"left_delimiter"`

	// RightDelim is the right delimiter for templating
	RightDelim *string `mapstructure:"right_delimiter"`
}

// DefaultDefaultsConfig returns the default DefaultsConfig
func DefaultDefaultsConfig() *DefaultsConfig {
	return &DefaultsConfig{}
}

// Copy returns a copy of the DefaultsConfig
func (c *DefaultsConfig) Copy() *DefaultsConfig {
	if c == nil {
		return nil
	}

	return &DefaultsConfig{
		LeftDelim:  c.LeftDelim,
		RightDelim: c.RightDelim,
	}
}

// Merge merges the DefaultsConfigs
func (c *DefaultsConfig) Merge(o *DefaultsConfig) *DefaultsConfig {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	if o.LeftDelim != nil {
		r.LeftDelim = o.LeftDelim
	}

	if o.RightDelim != nil {
		r.RightDelim = o.RightDelim
	}

	return r
}
