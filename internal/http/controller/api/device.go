package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	deps "github.com/lejianwen/rustdesk-api/v2/internal/http/deps"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/helper"
	requstform "github.com/lejianwen/rustdesk-api/v2/internal/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/internal/http/response"
	"github.com/lejianwen/rustdesk-api/v2/internal/model"
	"github.com/lejianwen/rustdesk-api/v2/internal/model/custom_types"
)

type Device struct {
	HD *deps.HandlerDeps
}

// Cli handles CLI-mode device registration/update.
// @Tags Device
// @Summary CLI device registration
// @Description Register or update device info via CLI mode
// @Accept  json
// @Produce  json
// @Param body body requstform.DeviceCliForm true "Device CLI form"
// @Success 200 {string} string ""
// @Failure 500 {object} response.ErrorResponse
// @Router /devices/cli [post]
// @Security BearerAuth
func (d *Device) Cli(c *gin.Context) {
	f := &requstform.DeviceCliForm{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	curUser := helper.CurUser(c)

	// Find or create the peer
	peer := d.HD.Services.FindById(f.Id)
	if peer == nil || peer.RowId == 0 {
		peer = &model.Peer{
			Id:     f.Id,
			Uuid:   f.Uuid,
			UserId: curUser.Id,
		}
		err = d.HD.Services.PeerService.Create(peer)
		if err != nil {
			response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
	}

	// Update peer fields
	if f.UserName != "" {
		u := d.HD.Services.InfoByUsername(f.UserName)
		if u != nil && u.Id != 0 {
			peer.UserId = u.Id
		}
	}

	if f.DeviceGroupName != "" {
		allGroups := d.HD.Services.DeviceGroupList(1, 999, nil)
		for _, g := range allGroups.DeviceGroups {
			if g.Name == f.DeviceGroupName {
				peer.GroupId = g.Id
				break
			}
		}
	}

	if f.StrategyName != "" {
		strategy := d.HD.Services.InfoByName(f.StrategyName)
		if strategy != nil && strategy.Id > 0 {
			if err := d.HD.Services.AssignToPeer(strategy.Id, peer.RowId); err != nil {
				d.HD.Logger.Warnf("AssignToPeer fail: strategy=%d peer=%d %v", strategy.Id, peer.RowId, err)
			}
		}
	}

	if f.Note != "" {
		peer.Note = f.Note
	}
	if f.DeviceUsername != "" {
		peer.Username = f.DeviceUsername
	}
	if f.DeviceName != "" {
		peer.Hostname = f.DeviceName
	}

	err = d.HD.Services.PeerService.Update(peer)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}

	// Handle address book assignment
	if f.AddressBookName != "" && peer.UserId != 0 {
		// Find or create peer in user's personal address book
		ab := d.HD.Services.InfoByUserIdAndIdAndCid(peer.UserId, peer.Id, 0)
		if ab == nil || ab.RowId == 0 {
			ab = &model.AddressBook{
				Id:       peer.Id,
				UserId:   peer.UserId,
				Username: peer.Username,
				Hostname: peer.Hostname,
				Platform: d.HD.Services.PlatformFromOs(peer.Os),
			}
		}
		if f.AddressBookAlias != "" {
			ab.Alias = f.AddressBookAlias
		}
		if f.AddressBookPassword != "" {
			ab.Password = f.AddressBookPassword
		}
		if f.AddressBookNote != "" {
			ab.Note = f.AddressBookNote
		}
		if f.AddressBookTag != "" {
			tagsJSON, _ := json.Marshal([]string{f.AddressBookTag})
			ab.Tags = custom_types.AutoJson(tagsJSON)
		}

		if ab.RowId == 0 {
			err = d.HD.Services.AddAddressBook(ab)
		} else {
			err = d.HD.Services.AddressBookService.Update(ab)
		}
		if err != nil {
			response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
	}

	c.String(http.StatusOK, "")
}

// Deploy handles `rustdesk --deploy` device provisioning (RustDesk 1.4.9+).
// It marks the device deployed, binds it to the authenticated user, reuses the
// presented token for the device, and returns a {"result": ...} verdict the
// client understands (OK / NOT_ENABLED / INVALID_INPUT / ID_TAKEN).
// @Tags Device
// @Summary Device deployment
// @Description Provision a device via `rustdesk --deploy`
// @Accept  json
// @Produce  json
// @Param body body requstform.DeviceDeployForm true "Device deploy form"
// @Success 200 {object} map[string]string
// @Router /devices/deploy [post]
// @Security BearerAuth
func (d *Device) Deploy(c *gin.Context) {
	if !d.HD.Config.Hbbs.DeployEnabled {
		c.JSON(http.StatusOK, gin.H{"result": "NOT_ENABLED"})
		return
	}
	f := &requstform.DeviceDeployForm{}
	if err := c.ShouldBindJSON(f); err != nil || f.Id == "" || f.Uuid == "" {
		c.JSON(http.StatusOK, gin.H{"result": "INVALID_INPUT"})
		return
	}
	curUser := helper.CurUser(c)
	if curUser == nil || curUser.Id == 0 {
		c.JSON(http.StatusOK, gin.H{"result": "INVALID_INPUT"})
		return
	}

	peer := d.HD.Services.FindById(f.Id)
	// The same id already claimed by a different machine.
	if peer != nil && peer.RowId != 0 && peer.Uuid != "" && peer.Uuid != f.Uuid {
		c.JSON(http.StatusOK, gin.H{"result": "ID_TAKEN"})
		return
	}

	if peer == nil || peer.RowId == 0 {
		peer = &model.Peer{Id: f.Id, Uuid: f.Uuid, UserId: curUser.Id, Deployed: true}
		if err := d.HD.Services.PeerService.Create(peer); err != nil {
			d.HD.Logger.Warnf("Deploy create peer fail: %v", err)
			c.JSON(http.StatusOK, gin.H{"result": "SERVER_ERROR"})
			return
		}
	} else {
		peer.Uuid = f.Uuid
		peer.UserId = curUser.Id
		peer.Deployed = true
		if err := d.HD.Services.PeerService.Update(peer); err != nil {
			d.HD.Logger.Warnf("Deploy update peer fail: %v", err)
			c.JSON(http.StatusOK, gin.H{"result": "SERVER_ERROR"})
			return
		}
	}

	// Reuse the presented access token for this device.
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	d.HD.Services.BindTokenToDevice(token, f.Uuid, f.Id)

	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}
