package sql

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ammo-collector/internal/repository/sql/dbmodel"
	"github.com/samber/mo"
)

type criteriaDTO struct {
	Handler        string `json:"handler"`
	HeaderCriteria []struct {
		HeaderName string `json:"headerName"`
		Pattern    string `json:"pattern"`
	} `json:"headerCriteria"`
}

// ConvertCollectionToEntity converts database Collection to an entity.Collection.
func ConvertCollectionToEntity(collection dbmodel.Collection) (entity.Collection, error) {
	var startedAt mo.Option[time.Time]
	if collection.StartedAt.Valid {
		startedAt = mo.Some(collection.StartedAt.Time)
	}

	var updatedAt mo.Option[time.Time]
	if collection.UpdatedAt.Valid {
		updatedAt = mo.Some(collection.UpdatedAt.Time)
	}

	var completedAt mo.Option[time.Time]
	if collection.CompletedAt.Valid {
		completedAt = mo.Some(collection.CompletedAt.Time)
	}

	var resultID mo.Option[entity.ResultID]
	if collection.ResultID.Valid {
		resultID = mo.Some(entity.ResultID(collection.ResultID.String))
	}

	var errorMessage mo.Option[string]
	if collection.ErrorMessage.Valid {
		errorMessage = mo.Some(collection.ErrorMessage.String)
	}

	var errorCode mo.Option[int]
	if collection.ErrorCode.Valid {
		errorCode = mo.Some(int(collection.ErrorCode.Int32))
	}

	var (
		task entity.Task
		err  error
	)
	if task, err = ConvertTaskToEntity(collection); err != nil {
		return entity.Collection{}, err
	}

	return entity.Collection{ //exhaustruct:enforce
		ID:           entity.CollectionID(collection.ID),
		Task:         task,
		Status:       entity.CollectionStatus(collection.Status),
		RequestCount: collection.RequestCount,
		CreatedAt:    collection.CreatedAt,
		StartedAt:    startedAt,
		UpdatedAt:    updatedAt,
		CompletedAt:  completedAt,
		ResultID:     resultID,
		ErrorMessage: errorMessage,
		ErrorCode:    errorCode,
	}, nil
}

// ConvertTaskToEntity converts database Collection to a Task.
func ConvertTaskToEntity(collection dbmodel.Collection) (entity.Task, error) {
	var dto criteriaDTO
	if err := json.Unmarshal(collection.Criteria, &dto); err != nil {
		// We can't return error here due to function signature, so we return empty Task
		return entity.Task{}, fmt.Errorf("convertTaskFromBytes: failed to unmarshal task to JSON: %w", err)
	}

	headerCriteria := make([]entity.HeaderCriteria, len(dto.HeaderCriteria))
	for i, hc := range dto.HeaderCriteria {
		pattern, err := regexp.Compile(hc.Pattern)
		if err != nil {
			return entity.Task{}, fmt.Errorf("convertTaskFromBytes:failed to compile regex: %w", err)
		}
		headerCriteria[i] = entity.HeaderCriteria{
			HeaderName: hc.HeaderName,
			Pattern:    pattern,
		}
	}

	return entity.Task{
		MessageSelection: entity.MessageSelectionCriteria{
			Handler:        dto.Handler,
			HeaderCriteria: headerCriteria,
		},
		Completion: entity.CompletionCriteria{
			TimeLimit:         collection.RequestDurationLimit,
			RequestCountLimit: collection.RequestCountLimit,
		},
	}, nil
}

// ConvertTaskToCriteriaDB converts Task to a criteriaDTO.
func ConvertTaskToCriteriaDB(task entity.Task) ([]byte, error) {
	dto := criteriaDTO{}
	dto.Handler = task.MessageSelection.Handler
	dto.HeaderCriteria = make([]struct {
		HeaderName string `json:"headerName"`
		Pattern    string `json:"pattern"`
	}, len(task.MessageSelection.HeaderCriteria))

	for i, hc := range task.MessageSelection.HeaderCriteria {
		dto.HeaderCriteria[i] = struct {
			HeaderName string `json:"headerName"`
			Pattern    string `json:"pattern"`
		}{
			HeaderName: hc.HeaderName,
			Pattern:    hc.Pattern.String(),
		}
	}
	data, err := json.Marshal(dto)
	if err != nil {
		return nil, fmt.Errorf("convertTaskToBytes: failed to marshal task to JSON: %w", err)
	}
	return data, nil
}
