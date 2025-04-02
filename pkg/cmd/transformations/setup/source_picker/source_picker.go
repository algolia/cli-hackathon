package source_picker

import (
	"fmt"
	"os"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	bubblelist "github.com/algolia/cli/pkg/cmd/transformations/bubble/list"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func PickSource(client *ingestion.APIClient) (string, error) {
	resp, err := client.ListSources(client.NewApiListSourcesRequest().WithType([]ingestion.SourceType{ingestion.SOURCE_TYPE_JSON, ingestion.SOURCE_TYPE_CSV, ingestion.SOURCE_TYPE_DOCKER, ingestion.SOURCE_TYPE_BIGQUERY}).WithItemsPerPage(100))
	if err != nil {
		return "", err
	}

	items := make([]list.Item, 0, len(resp.Sources))

	for _, source := range resp.Sources {
		items = append(items, bubblelist.Item{Name: fmt.Sprintf("%s: %s", source.GetType(), source.GetName()), UUID: source.GetSourceID()})
	}

	list := bubblelist.NewBubbleList("sources", items)

	if _, err := tea.NewProgram(&list).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return list.Choice, nil
}
