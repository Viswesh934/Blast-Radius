package commands

import (
	"fmt"
	"strings"

	"github.com/Viswesh934/blast-radius/internal/openmetadata"
	"go.uber.org/zap"
)

func newOMClient() (*openmetadata.Client, error) {
	if strings.TrimSpace(cfg.OpenMetadata.BaseURL) == "" {
		return nil, fmt.Errorf("openmetadata.baseurl is required (set OM_BASE_URL or config)")
	}
	return openmetadata.NewClient(cfg.OpenMetadata.BaseURL, cfg.OpenMetadata.JWTToken, zap.NewNop()), nil
}
