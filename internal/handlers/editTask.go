package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"go_final_project/internal/models"
)

func (h *Handler) EditTask(w http.ResponseWriter, r *http.Request) {
	var task models.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		err = fmt.Errorf("can't parse response")
		h.SendErr(w, err, http.StatusBadRequest)
		return
	}
	err = task.CheckTask()
	if err != nil {
		h.SendErr(w, err, http.StatusBadRequest)
		return
	} else {
		id, err := strconv.Atoi(task.ID)
		if err != nil {
			err = fmt.Errorf("cat not parse id")
			h.SendErr(w, err, http.StatusBadRequest)
			return
		}

		err = h.db.CheckID(id)
		if err != nil {
			h.SendErr(w, err, http.StatusBadRequest)
			return
		}
		task.Date, err = task.CheckDate()
		if err != nil {
			h.SendErr(w, err, http.StatusBadRequest)
			return
		}
	}
	err = h.db.Update(&task)
	if err != nil {
		h.SendErr(w, err, http.StatusInternalServerError)
		return
	}
	h.logger.Infof("sent response via handler Task (method %s)", r.Method)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	_, err = w.Write([]byte("{}"))
	if err != nil {
		h.logger.Error(err)
	}
}
