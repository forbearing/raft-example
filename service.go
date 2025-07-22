package main

import (
	"net/http"

	"demo/types"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type service struct {
	s types.Store
}

type JoinReq struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}
type SetReq map[string]string

func (s *service) Set(c *gin.Context) {
	req := SetReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Error(err)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	for k, v := range req {
		s.s.Set(k, v)
	}

	c.String(http.StatusOK, "ok")
}

func (s *service) Get(c *gin.Context) {
	key := c.Param("key")
	v, err := s.s.Get(key)
	if err != nil {
		logrus.Error(err)
		c.String(http.StatusNotFound, err.Error())
		return
	}
	c.String(http.StatusOK, v)
}

func (s *service) Join(c *gin.Context) {
	req := new(JoinReq)
	if err := c.ShouldBindJSON(req); err != nil {
		logrus.Error(err)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	if err := s.s.Join(req.ID, req.Addr); err != nil {
		logrus.Error(err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.String(http.StatusOK, "ok")
}

func (s *service) Status(c *gin.Context) {
	status, err := s.s.Status()
	if err != nil {
		logrus.Error(err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, status)
}
