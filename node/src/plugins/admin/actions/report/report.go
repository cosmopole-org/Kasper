package actions_report

import (
	"encoding/json"
	"kasper/src/abstract"
	admin_inputs_report "kasper/src/plugins/admin/inputs/report"
	admin_model "kasper/src/plugins/admin/model"
	admin_outputs_report "kasper/src/plugins/admin/outputs/report"
	"kasper/src/shell/api/model"
	"kasper/src/shell/layer1/adapters"
	module_state "kasper/src/shell/layer1/module/state"
	"log"

	"gorm.io/datatypes"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return s.AutoMigrate(&admin_model.Report{})
}

// ReadReports /admin/report/read check [ true false false ] access [ true false false false PUT ]
func (a *Actions) ReadReports(s abstract.IState, input admin_inputs_report.ReadReportsInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	trx := state.Trx()
	reports := []admin_model.Report{}
	trx.Db().Find(&reports)
	messageIds := []string{}
	for _, report := range reports {
		str, err := json.Marshal(report.Data)
		if err != nil {
			log.Println(err)
			continue
		}
		data := map[string]any{}
		err2 := json.Unmarshal(str, &data)
		if err2 != nil {
			log.Println(err2)
			continue
		}
		msgIdRaw, ok := data["messageId"]
		if !ok {
			continue
		}
		msgId, ok2 := msgIdRaw.(string)
		if !ok2 {
			continue
		}
		messageIds = append(messageIds, msgId)
	}
	userIds := []string{}
	for _, report := range reports {
		str, err := json.Marshal(report.Data)
		if err != nil {
			log.Println(err)
			continue
		}
		data := map[string]any{}
		err2 := json.Unmarshal(str, &data)
		if err2 != nil {
			log.Println(err2)
			continue
		}
		userIdRaw, ok := data["userId"]
		if !ok {
			continue
		}
		userId, ok2 := userIdRaw.(string)
		if !ok2 {
			continue
		}
		userIds = append(userIds, userId)
	}
	messages := []admin_model.Message{}
	trx.Db().Model(&admin_model.Message{}).Where("id in ?", messageIds).Find(&messages)
	messageDict := map[string]*admin_model.ResultMessage{}
	authorIds := []string{}
	for _, msg := range messages {
		authorIds = append(authorIds, msg.AuthorId)
	}
	type author struct {
		Id      string
		Profile datatypes.JSON
	}
	authors := []author{}
	trx.Db().Model(&model.User{}).Select("id as id, "+adapters.BuildJsonFetcher("metadata", "hokm.profile")+" as profile").Where("id in ?", authorIds).Find(&authors)
	authorDict := map[string]author{}
	for _, a := range authors {
		authorDict[a.Id] = a
	}
	for _, msg := range messages {
		messageDict[msg.Id] = &admin_model.ResultMessage{
			Id:       msg.Id,
			SpaceId:  msg.SpaceId,
			TopicId:  msg.TopicId,
			AuthorId: msg.AuthorId,
			Data:     msg.Data,
			Time:     msg.Time,
			Author:   authorDict[msg.AuthorId].Profile,
		}
	}

	type user struct {
		Id      string
		Profile datatypes.JSON
	}
	users := []user{}
	trx.Db().Model(&model.User{}).Select("id as id, "+adapters.BuildJsonFetcher("metadata", "hokm.profile")+" as profile").Where("id in ?", userIds).Find(&users)
	usersDict := map[string]user{}
	for _, u := range users {
		usersDict[u.Id] = u
	}

	result := []admin_model.ResultReport{}
	for _, report := range reports {
		str, err := json.Marshal(report.Data)
		if err != nil {
			log.Println(err)
			continue
		}
		data := map[string]any{}
		err2 := json.Unmarshal(str, &data)
		if err2 != nil {
			log.Println(err2)
			continue
		}
		msgIdRaw, ok := data["messageId"]
		userIdRaw, ok2 := data["userId"]
		if ok {
			msgId, ok3 := msgIdRaw.(string)
			if !ok3 {
				continue
			}
			data["message"] = messageDict[msgId]
			result = append(result, admin_model.ResultReport{
				Id:         report.Id,
				ReporterId: report.ReporterId,
				GameKey:    report.GameKey,
				Data:       data,
			})
		} else if ok2 {
			userId, ok3 := userIdRaw.(string)
			if !ok3 {
				continue
			}
			data["user"] = usersDict[userId]
			result = append(result, admin_model.ResultReport{
				Id:         report.Id,
				ReporterId: report.ReporterId,
				GameKey:    report.GameKey,
				Data:       data,
			})
		}
	}
	return admin_outputs_report.ReadReportsOutput{Reports: result}, nil
}

// ResolveReport /admin/report/resolve check [ true false false ] access [ true false false false PUT ]
func (a *Actions) ResolveReport(s abstract.IState, input admin_inputs_report.ResolveReportsInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	trx := state.Trx()
	report := admin_model.Report{Id: input.ReportId}
	trx.Db().Delete(&report)
	return admin_outputs_report.ResolveReportOutput{}, nil
}

// ClearReports /admin/report/clear check [ true false false ] access [ true false false false PUT ]
func (a *Actions) ClearReports(s abstract.IState, input admin_inputs_report.ClearReportsInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	trx := state.Trx()
	trx.Db().Where("1 = 1").Delete(&admin_model.Report{})
	return admin_outputs_report.ResolveReportOutput{}, nil
}
