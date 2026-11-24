package v2

import (
	"github.com/1Panel-dev/1Panel/core/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/gin-gonic/gin"
)

// @Tags System Group
// @Summary Create group
// @Accept json
// @Param request body dto.GroupCreate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/groups [post]
// @x-panel-log {"bodyKeys":["name","type"],"paramKeys":[],"BeforeFunctions":[],"formatEN":"create group [name][type]"}
func (b *BaseApi) CreateGroup(c *gin.Context) {
	var req dto.GroupCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := groupService.Create(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Group
// @Summary Delete group
// @Accept json
// @Param request body dto.OperateByID true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/groups/del [post]
// @x-panel-log {"bodyKeys":["id"],"paramKeys":[],"BeforeFunctions":[{"input_column":"id","input_value":"id","isList":false,"db":"groups","output_column":"name","output_value":"name"},{"input_column":"id","input_value":"id","isList":false,"db":"groups","output_column":"type","output_value":"type"}],"formatEN":"delete group [type][name]"}
func (b *BaseApi) DeleteGroup(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := groupService.Delete(req.ID); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Group
// @Summary Update group
// @Accept json
// @Param request body dto.GroupUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/groups/update [post]
// @x-panel-log {"bodyKeys":["name","type"],"paramKeys":[],"BeforeFunctions":[],"formatEN":"update group [name][type]"}
func (b *BaseApi) UpdateGroup(c *gin.Context) {
	var req dto.GroupUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := groupService.Update(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Group
// @Summary List groups
// @Accept json
// @Param request body dto.GroupSearch true "request"
// @Success 200 {array} dto.OperateByType
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/groups/search [post]
func (b *BaseApi) ListGroup(c *gin.Context) {
	var req dto.OperateByType
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	list, err := groupService.List(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}

	helper.SuccessWithData(c, list)
}
