package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/user"
	"github.com/gin-gonic/gin"
)

// GroupShareList lists all group share areas with the current user's relationship to each.
func GroupShareList(c *gin.Context) {
	res, err := user.ListGroupShares(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// GroupShareApply submits a request to join a group share area.
func GroupShareApply(c *gin.Context) {
	service := ParametersFromContext[*user.ApplyGroupShareService](c, user.ApplyGroupShareParamCtx{})
	service.ID = c.Param("id")
	if err := service.Apply(c); err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}
	c.JSON(200, serializer.Response{})
}

// GroupShareApplications lists pending applicants of a group share area (approver only).
func GroupShareApplications(c *gin.Context) {
	service := ParametersFromContext[*user.GroupShareIDService](c, user.GroupShareIDParamCtx{})
	res, err := service.ListApplications(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// GroupShareReview approves or rejects a pending applicant (approver only).
func GroupShareReview(c *gin.Context) {
	service := ParametersFromContext[*user.ReviewGroupShareService](c, user.ReviewGroupShareParamCtx{})
	service.ID = c.Param("id")
	if err := service.Review(c); err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}
	c.JSON(200, serializer.Response{})
}
