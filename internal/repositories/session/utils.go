package session

import (
	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/ethanbaker/assistant/pkg/logger"
	"github.com/openai/openai-go/v2/shared/constant"
)

// Helper function to sort tool calls into tool call/tool call response pairs
func sortItemsPrePersist(items []*domain.Item) []*domain.Item {
	calls := map[string]*domain.Item{}
	resps := map[string]*domain.Item{}

	callIndex := -1

	// Separate calls and responses by id
	for i, item := range items {
		callId, isToolCall := getToolCallIdFromInput(item.ResponseItem)
		if isToolCall {
			if callIndex == -1 {
				callIndex = i
			}

			calls[callId] = item
			continue
		}

		respId, isToolResp := getToolCallIdFromOutput(item.ResponseItem)
		if isToolResp {
			resps[respId] = item
		}
	}

	// No tool calls, return items
	if callIndex == -1 {
		return items
	}

	// Add non-tool call items to sorted at the front
	sorted := []*domain.Item{}
	sorted = append(sorted, items[:callIndex]...)

	// Reconstruct ordering in pairs
	for id, call := range calls {
		resp, ok := resps[id]
		if !ok {
			logger.Errorf("Tool call with id %s exists with no response", id)
			continue
		}

		sorted = append(sorted, call, resp)
	}

	// Log warnings for orphaned responses
	for id := range resps {
		_, ok := calls[id]
		if !ok {
			logger.Errorf("Tool call response with id %s exists with no call", id)
		}
	}

	// Add non-tool call items to sorted at the back
	sorted = append(sorted, items[callIndex+len(calls)*2:]...)

	return sorted
}

// Helper function to get tool call id from tool call input
func getToolCallIdFromInput(embeddedItem domain.ResponseItemData) (string, bool) {
	item := embeddedItem.TResponseInputItem

	switch *item.GetType() {
	case string(constant.ValueOf[constant.FunctionCall]()):
		return item.OfFunctionCall.CallID, true
	case string(constant.ValueOf[constant.LocalShellCall]()):
		return item.OfLocalShellCall.CallID, true
	case string(constant.ValueOf[constant.CustomToolCall]()):
		return item.OfCustomToolCall.CallID, true
	default:
		return "", false
	}
}

// Helper function to get tool call id from tool call output
func getToolCallIdFromOutput(embeddedItem domain.ResponseItemData) (string, bool) {
	item := embeddedItem.TResponseInputItem

	switch *item.GetType() {
	case string(constant.ValueOf[constant.FunctionCallOutput]()):
		return item.OfFunctionCallOutput.CallID, true
	case string(constant.ValueOf[constant.ComputerCallOutput]()):
		return item.OfComputerCallOutput.CallID, true
	case string(constant.ValueOf[constant.LocalShellCallOutput]()):
		return item.OfLocalShellCallOutput.ID, true
	case string(constant.ValueOf[constant.CustomToolCallOutput]()):
		return item.OfCustomToolCallOutput.CallID, true
	default:
		return "", false
	}
}
