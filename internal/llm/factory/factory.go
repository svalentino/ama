package factory

import (
	"fmt"

	"github.com/svalentino/ama/internal/config"
	"github.com/svalentino/ama/internal/llm"
	"github.com/svalentino/ama/internal/llm/bedrock"
)

func New(current config.Config) (llm.Provider, error) {
	switch current.ProviderName() {
	case config.ProviderBedrock:
		provider, err := bedrock.New(bedrock.Config{
			Profile: current.ProviderOption(config.ProviderOptionProfile),
			Region:  current.ProviderOption(config.ProviderOptionRegion),
		})
		if err != nil {
			return nil, fmt.Errorf("initialize provider %q: %w", current.ProviderName(), err)
		}
		return provider, nil
	default:
		return nil, fmt.Errorf("provider %q is not supported", current.ProviderName())
	}
}
