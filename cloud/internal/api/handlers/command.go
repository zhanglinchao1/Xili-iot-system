package handlers

import (
	"fmt"
	"net/http"

	"cloud-system/internal/models"
	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/gin-gonic/gin"
)

// CommandHandler 命令处理器
type CommandHandler struct {
	commandService services.CommandService
}

// NewCommandHandler 创建命令处理器实例
func NewCommandHandler(commandService services.CommandService) *CommandHandler {
	return &CommandHandler{
		commandService: commandService,
	}
}

// SendCommand 发送命令到Edge端
// @Summary 发送命令
// @Tags Command
// @Accept json
// @Produce json
// @Param cabinet_id path string true "储能柜ID"
// @Param request body models.SendCommandRequest true "命令请求"
// @Success 200 {object} utils.SuccessResponse{data=models.Command}
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/commands/{cabinet_id} [post]
func (h *CommandHandler) SendCommand(c *gin.Context) {
	cabinetID := c.Param("cabinet_id")

	var request models.SendCommandRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	// 从上下文获取用户信息（通过JWT中间件设置）
	createdBy := "admin"
	if user, exists := c.Get("user_id"); exists {
		// user_id 可能是 int 或 string 类型
		switch v := user.(type) {
		case int:
			createdBy = fmt.Sprintf("%d", v)
		case int64:
			createdBy = fmt.Sprintf("%d", v)
		case string:
			createdBy = v
		default:
			createdBy = fmt.Sprintf("%v", v)
		}
	}

	command, err := h.commandService.SendCommand(c.Request.Context(), cabinetID, &request, createdBy)
	if err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		if appErr.Code == errors.ErrCabinetNotFound {
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, command, "命令已发送")
}

// GetCommand 获取命令详情
// @Summary 获取命令详情
// @Tags Command
// @Accept json
// @Produce json
// @Param command_id path string true "命令ID"
// @Success 200 {object} utils.SuccessResponse{data=models.Command}
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/commands/{command_id} [get]
func (h *CommandHandler) GetCommand(c *gin.Context) {
	commandID := c.Param("command_id")

	command, err := h.commandService.GetCommand(c.Request.Context(), commandID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.(*errors.AppError))
		return
	}

	utils.Success(c, command)
}

// ListCommands 获取命令列表
// @Summary 获取命令列表
// @Tags Command
// @Accept json
// @Produce json
// @Param cabinet_id query string false "储能柜ID过滤"
// @Param command_type query string false "命令类型过滤"
// @Param status query string false "状态过滤"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} utils.PaginatedResponse{data=[]models.Command}
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/commands [get]
func (h *CommandHandler) ListCommands(c *gin.Context) {
	var filter models.CommandListFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationError(c, "查询参数格式错误")
		return
	}

	commands, total, err := h.commandService.ListCommands(c.Request.Context(), &filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.(*errors.AppError))
		return
	}

	utils.SuccessPaginated(c, commands, filter.Page, filter.PageSize, total)
}

// AckCommand Edge端回执命令状态
func (h *CommandHandler) AckCommand(c *gin.Context) {
	commandID := c.Param("command_id")

	apiKeyValue, exists := c.Get("edge_api_key")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, errors.NewUnauthorizedError("缺少API Key"))
		return
	}
	apiKey := apiKeyValue.(string)

	var req models.CommandAckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "请求参数格式错误")
		return
	}

	if err := h.commandService.AckCommand(c.Request.Context(), commandID, apiKey, req.Status, req.Message); err != nil {
		appErr := err.(*errors.AppError)
		statusCode := http.StatusBadRequest
		switch appErr.Code {
		case errors.ErrUnauthorized:
			statusCode = http.StatusUnauthorized
		case errors.ErrForbidden:
			statusCode = http.StatusForbidden
		case errors.ErrNotFound:
			statusCode = http.StatusNotFound
		}
		utils.ErrorResponse(c, statusCode, appErr)
		return
	}

	utils.SuccessWithMessage(c, nil, "命令状态已更新")
}
