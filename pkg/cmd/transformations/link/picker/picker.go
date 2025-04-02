package picker

import (
	"fmt"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	bubblelist "github.com/algolia/cli/pkg/cmd/transformations/bubble/list"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func cloneDestination(client *ingestion.APIClient, destinationID string) (string, error) {
	destination, err := client.GetDestination(client.NewApiGetDestinationRequest(destinationID))
	if err != nil {
		return "", err
	}

	destinationCreate := ingestion.NewDestinationCreate(
		destination.GetType(),
		fmt.Sprintf("%s (clone - %s)", destination.GetName(), time.Now().Format(time.RFC3339Nano)),
		destination.GetInput(),
	)

	if authID, ok := destination.GetAuthenticationIDOk(); ok {
		destinationCreate.SetAuthenticationID(*authID)
	}

	resp, err := client.CreateDestination(client.NewApiCreateDestinationRequest(destinationCreate))
	if err != nil {
		return "", err
	}

	return resp.GetDestinationID(), nil
}

func selectDestination(items []list.Item) (string, error) {
	destinations := bubblelist.NewBubbleList("destinations", items)

	if _, err := tea.NewProgram(&destinations).Run(); err != nil {
		return "", err
	}

	return destinations.Choice, nil
}

func PickDestination(client *ingestion.APIClient) (string, error) {
	resp, err := client.ListDestinations(
		client.
			NewApiListDestinationsRequest().
			WithType([]ingestion.DestinationType{ingestion.DESTINATION_TYPE_SEARCH}).
			WithItemsPerPage(100),
	)
	if err != nil {
		return "", err
	}

	items := make([]list.Item, 0, len(resp.Destinations))
	hasTransformationID := make(map[string]bool, len(resp.Destinations))

	for _, destination := range resp.GetDestinations() {
		indexName, hasIndexName := destination.Input.DestinationIndexName.GetIndexNameOk()

		itemName := destination.GetName()
		if hasIndexName {
			itemName = fmt.Sprintf("%s: %s", *indexName, itemName)
		}

		if len(destination.GetTransformationIDs()) > 0 {
			itemName = fmt.Sprintf("%s (linked)", itemName)
			hasTransformationID[destination.GetDestinationID()] = true
		}

		items = append(items, bubblelist.Item{
			Name: itemName,
			UUID: destination.GetDestinationID(),
		})
	}

	choice, err := selectDestination(items)
	if err != nil {
		return "", err
	}

	if hasTransformationID[choice] {
		decisions := []bubblelist.Item{
			{Name: "Replace the existing transformation with the new one", UUID: string(Replace)},
			{Name: "Clone the destination and attach the new transformation to it", UUID: string(Clone)},
			{Name: "Keep the existing transformation and quit the program", UUID: string(Keep)},
		}

		decision, err := Decide(
			"destination already has a transformation linked. Do you still want to link a new transformation?",
			decisions,
		)
		if err != nil {
			return "", err
		}

		switch decision {
		case Replace:
			// Link the new transformation to the destination
			// `choice` is already correct
		case Clone:
			// Create a new destination, and link the new transformation to it
			newDestinationID, err := cloneDestination(client, choice)
			if err != nil {
				return "", err
			}

			choice = newDestinationID
		case Keep:
			return "", nil
		}
	}

	// We check the user knows that he is aiming to updating a destination already attached to tasks
	res, err := client.ListTasks(client.NewApiListTasksRequest().WithDestinationID([]string{choice}))
	if err != nil {
		return "", err
	}

	if len(res.GetTasks()) > 0 {
		decisions := []bubblelist.Item{
			{Name: "Replace the existing transformation in the destination (will affect existing tasks!)", UUID: string(Replace)},
			{Name: "Clone the destination and attach the transformation to the new destination", UUID: string(Clone)},
			{Name: "Don't link the transformation and quit the program", UUID: string(Keep)},
		}
		decision, err := Decide(
			"destination is attached to some tasks. If you link a new transformation, the existing tasks will be affected. What do you want to do?",
			decisions,
		)
		if err != nil {
			return "", err
		}

		switch decision {
		case Replace:
			return choice, nil
		case Keep:
			// Quit the program
			return "", nil
		case Clone:
			// Create a new destination, and link the new transformation to it
			return cloneDestination(client, choice)
		}
	}

	return choice, nil
}
