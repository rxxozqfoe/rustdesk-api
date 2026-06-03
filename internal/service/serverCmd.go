package service

import (
	"fmt"
	"net"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/internal/model"
)

type ServerCmdService struct {
	ctx *ServiceContext
}

// List
func (is *ServerCmdService) List(page, pageSize uint) *model.ServerCmdList {
	res := &model.ServerCmdList{}
	queryList(is.ctx.DB, page, pageSize, res, &res.ServerCmds, nil)
	return res
}

// Info
func (is *ServerCmdService) Info(id uint) *model.ServerCmd {
	u := &model.ServerCmd{}
	is.ctx.DB.Where("id = ?", id).First(u)
	return u
}

// Delete
func (is *ServerCmdService) Delete(u *model.ServerCmd) error {
	return is.ctx.DB.Delete(u).Error
}

// Create
func (is *ServerCmdService) Create(u *model.ServerCmd) error {
	res := is.ctx.DB.Create(u).Error
	return res
}

// SendCmd 发送命令
func (is *ServerCmdService) SendCmd(port int, cmd string, arg string) (string, error) {
	//组装命令
	cmd = cmd + " " + arg
	res, err := is.SendSocketCmd("v6", port, cmd)
	if err == nil {
		return res, nil
	}
	//v6连接失败，尝试v4
	res, err = is.SendSocketCmd("v4", port, cmd)
	if err == nil {
		return res, nil
	}
	return "", err
}

// SendSocketCmd
func (is *ServerCmdService) SendSocketCmd(ty string, port int, cmd string) (string, error) {
	addr := "[::1]"
	tcp := "tcp6"
	if ty == "v4" {
		tcp = "tcp"
		addr = "127.0.0.1"
	}
	conn, err := net.Dial(tcp, fmt.Sprintf("%s:%v", addr, port))
	if err != nil {
		is.ctx.Logger.Debugf("%s connect to id server failed: %v", ty, err)
		return "", err
	}
	defer func() { _ = conn.Close() }()
	//发送命令
	_, err = conn.Write([]byte(cmd))
	if err != nil {
		is.ctx.Logger.Debugf("%s send cmd failed: %v", ty, err)
		return "", err
	}
	time.Sleep(100 * time.Millisecond)
	//读取返回
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err.Error() != "EOF" {
		is.ctx.Logger.Debugf("%s read response failed: %v", ty, err)
		return "", err
	}
	return string(buf[:n]), nil
}

func (is *ServerCmdService) Update(f *model.ServerCmd) error {
	return is.ctx.DB.Model(f).Updates(f).Error
}
