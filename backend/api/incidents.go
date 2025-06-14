package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"pluralkit/status/util"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (a *API) GetIncidents(w http.ResponseWriter, r *http.Request) {
	var err error
	timeText := r.URL.Query().Get("before")
	var before time.Time
	if len(timeText) == 0 {
		before = time.Now()
	} else {
		before, err = time.Parse(time.RFC3339, timeText)
		if err != nil {
			http.Error(w, "error while parsing 'before' argument", 400)
			return
		}
	}

	list, err := a.Database.GetIncidentsBefore(r.Context(), before)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while fufilling incidents request", slog.Any("error", err))
		return
	}

	if err := render.Render(w, r, &list); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while rendering json for active incidents request", slog.Any("error", err))
		return
	}
}

func (a *API) GetActiveIncidents(w http.ResponseWriter, r *http.Request) {
	list, err := a.Database.GetActiveIncidents(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while fufilling active incidents request", slog.Any("error", err))
		return
	}

	if err := render.Render(w, r, &list); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while rendering json for active incidents request", slog.Any("error", err))
		return
	}
}

func (a *API) GetIncident(w http.ResponseWriter, r *http.Request) {
	incident, err := a.Database.GetIncident(r.Context(), chi.URLParam(r, "incidentID"))
	if err != nil {
		if errors.Is(err, util.ErrInvalid) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if errors.Is(err, util.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while fufilling get incident request", slog.Any("error", err))
		return
	}
	if err := render.Render(w, r, &incident); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while rendering json for get incident request", slog.Any("error", err))
		return
	}
}

func (a *API) CreateIncident(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while getting body data", slog.Any("error", err))
		return
	}

	var incident util.Incident

	err = json.Unmarshal(data, &incident)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		a.Logger.Error("error while parsing incident data", slog.Any("error", err))
		return
	}

	id, err := a.Database.CreateIncident(r.Context(), incident)
	if err != nil {
		if errors.Is(err, util.ErrInvalid) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while creating incident", slog.Any("error", err))
		return
	}

	_, err = w.Write([]byte(id))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while sending response", slog.Any("error", err))
	}
}

func (a *API) EditIncident(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while getting body data", slog.Any("error", err))
		return
	}

	var incidentPatch util.IncidentPatch
	id := chi.URLParam(r, "incidentID")

	err = json.Unmarshal(data, &incidentPatch)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		a.Logger.Error("error while parsing incident data", slog.Any("error", err))
		return
	}

	err = a.Database.EditIncident(r.Context(), id, incidentPatch)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if errors.Is(err, util.ErrInvalid) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while editing incident", slog.Any("error", err))
		return
	}
}

func (a *API) DeleteIncident(w http.ResponseWriter, r *http.Request) {
	var incident util.Incident
	incident.ID = chi.URLParam(r, "incidentID")

	err := a.Database.DeleteIncident(r.Context(), incident)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if errors.Is(err, util.ErrInvalid) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while deleting incident", slog.Any("error", err))
		return
	}
}

func (a *API) AddUpdate(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while getting body data", slog.Any("error", err))
		return
	}

	var update util.IncidentUpdate
	err = json.Unmarshal(data, &update)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		a.Logger.Error("error while parsing update data", slog.Any("error", err))
		return
	}
	update.IncidentID = chi.URLParam(r, "incidentID")

	id, err := a.Database.CreateUpdate(r.Context(), update)
	if err != nil {
		if errors.Is(err, util.ErrInvalid) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if errors.Is(err, util.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while creating update", slog.Any("error", err))
		return
	}

	_, err = w.Write([]byte(id))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while sending response", slog.Any("error", err))
	}
}

func (a *API) EditUpdate(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while getting body data", slog.Any("error", err))
		return
	}

	var update util.UpdatePatch
	id := chi.URLParam(r, "updateID")
	str := string(data)
	update.Text = &str

	err = a.Database.EditUpdate(r.Context(), id, update)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if errors.Is(err, util.ErrInvalid) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while editing update", slog.Any("error", err))
		return
	}
}

func (a *API) GetUpdate(w http.ResponseWriter, r *http.Request) {
	update, err := a.Database.GetUpdate(r.Context(), chi.URLParam(r, "updateID"))
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if errors.Is(err, util.ErrInvalid) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while getting update", slog.Any("error", err))
		return
	}

	if err := render.Render(w, r, &update); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while rendering json for get incident request", slog.Any("error", err))
		return
	}
}

func (a *API) DeleteUpdate(w http.ResponseWriter, r *http.Request) {
	var update util.IncidentUpdate
	update.IncidentID = chi.URLParam(r, "incidentID")
	update.ID = chi.URLParam(r, "updateID")

	err := a.Database.DeleteUpdate(r.Context(), update)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if errors.Is(err, util.ErrInvalid) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		a.Logger.Error("error while deleting update", slog.Any("error", err))
		return
	}
}
