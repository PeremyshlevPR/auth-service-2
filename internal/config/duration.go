package config

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Duration extends time.Duration to support "d" (days) suffix
type Duration struct {
	time.Duration
}

// EnvDecode implements envconfig.Decoder to parse duration with days support
func (d *Duration) EnvDecode(ctx context.Context, v string) error {
	if v == "" {
		return nil
	}

	// Check if the value ends with 'd' for days
	if strings.HasSuffix(v, "d") {
		daysStr := strings.TrimSuffix(v, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return fmt.Errorf("invalid days value: %w", err)
		}
		d.Duration = time.Duration(days) * 24 * time.Hour
		return nil
	}

	// Otherwise, parse as standard duration
	duration, err := time.ParseDuration(v)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}
	d.Duration = duration
	return nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (d *Duration) UnmarshalText(text []byte) error {
	return d.EnvDecode(context.Background(), string(text))
}

// MarshalText implements encoding.TextMarshaler
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}

// String returns the string representation of the duration
func (d Duration) String() string {
	return d.Duration.String()
}
