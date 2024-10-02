package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/Hobrus/hobrusmetrics.git/internal/repositories"
	"strings"

	"github.com/Hobrus/hobrusmetrics.git/internal/service"
	"github.com/valyala/fasthttp"
)

type Router struct {
	metricsService *service.MetricsService
}

func NewRouter(ms *service.MetricsService) fasthttp.RequestHandler {
	r := &Router{metricsService: ms}
	return r.handle
}

func (r *Router) handle(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/":
		r.handleRoot(ctx)
	case "/update/":
		r.handleJSONUpdate(ctx)
	case "/value/":
		r.handleJSONValue(ctx)
	default:
		if strings.HasPrefix(string(ctx.Path()), "/update/") {
			r.handleUpdate(ctx)
		} else if strings.HasPrefix(string(ctx.Path()), "/value/") {
			r.handleValue(ctx)
		} else {
			ctx.Error("Not found", fasthttp.StatusNotFound)
		}
	}
}

func (r *Router) handleUpdate(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		ctx.Error("Method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(string(ctx.Path())[1:], "/")
	if len(parts) != 4 || parts[0] != "update" {
		ctx.Error("Not found", fasthttp.StatusNotFound) // Изменено с BadRequest на NotFound
		return
	}

	metricType := parts[1]
	metricName := parts[2]
	metricValue := parts[3]

	if metricName == "" {
		ctx.Error("Not found", fasthttp.StatusNotFound) // Добавлена проверка на пустое имя метрики
		return
	}

	err := r.metricsService.UpdateMetric(metricType, metricName, metricValue)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (r *Router) handleValue(ctx *fasthttp.RequestCtx) {
	parts := strings.Split(string(ctx.Path())[1:], "/")
	if len(parts) != 3 || parts[0] != "value" {
		ctx.Error("Invalid URL format", fasthttp.StatusBadRequest)
		return
	}

	metricType := parts[1]
	metricName := parts[2]

	value, err := r.metricsService.GetMetricValue(metricType, metricName)
	if err != nil {
		ctx.Error("Metric not found", fasthttp.StatusNotFound)
		return
	}

	ctx.SetContentType("text/plain")
	fmt.Fprintf(ctx, "%v", value)
}

func (r *Router) handleRoot(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/html")
	fmt.Fprintf(ctx, "<html><body><h1>Metrics</h1><ul>")

	for name, value := range r.metricsService.GetAllMetrics() {
		fmt.Fprintf(ctx, "<li>%s: %v</li>", name, value)
	}

	fmt.Fprintf(ctx, "</ul></body></html>")
}

type MetricRequest struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type MetricResponse struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

func (r *Router) handleJSONUpdate(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		ctx.Error("Method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var req MetricRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error("Invalid JSON", fasthttp.StatusBadRequest)
		return
	}

	if req.ID == "" {
		ctx.Error("Not found", fasthttp.StatusNotFound) // Добавлена проверка на пустой ID
		return
	}

	var err error
	switch req.MType {
	case "gauge":
		if req.Value == nil {
			ctx.Error("Value is required for gauge", fasthttp.StatusBadRequest)
			return
		}
		err = r.metricsService.UpdateMetric(req.MType, req.ID, fmt.Sprintf("%f", *req.Value))
	case "counter":
		if req.Delta == nil {
			ctx.Error("Delta is required for counter", fasthttp.StatusBadRequest)
			return
		}
		err = r.metricsService.UpdateMetric(req.MType, req.ID, fmt.Sprintf("%d", *req.Delta))
	default:
		ctx.Error("Unknown metric type", fasthttp.StatusBadRequest)
		return
	}

	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (r *Router) handleJSONValue(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		ctx.Error("Method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var req MetricRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error("Invalid JSON", fasthttp.StatusBadRequest)
		return
	}

	value, err := r.metricsService.GetMetricValue(req.MType, req.ID)
	if err != nil {
		ctx.Error("Metric not found", fasthttp.StatusNotFound)
		return
	}

	resp := MetricResponse{
		ID:    req.ID,
		MType: req.MType,
	}

	switch req.MType {
	case "gauge":
		floatValue := float64(value.(repositories.Gauge))
		resp.Value = &floatValue
	case "counter":
		intValue := int64(value.(repositories.Counter))
		resp.Delta = &intValue
	default:
		ctx.Error("Unknown metric type", fasthttp.StatusBadRequest)
		return
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		ctx.Error("Error encoding response", fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetBody(jsonResp)
}
