package goToolSVRZ5

import (
	"errors"
	"fmt"
	"github.com/Deansquirrel/goToolCommon"
	"github.com/Deansquirrel/goToolEnvironment"
	"github.com/Deansquirrel/goToolMSSql"
	"github.com/Deansquirrel/goToolMSSql2000"
	"github.com/Deansquirrel/goToolMSSqlHelper"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	sqlGetMdAccount = "" +
		"select accname " +
		"from %s " +
		"where accisdeleted = 0 " +
		"	and acctype = 1"
	sqlGetZlCompany = "" +
		"SELECT [coid],[coab],[cocode],[cotype],[couserab]," +
		"	[cousercode],[coacccrdate] " +
		"FROM [zlcompany]"
	sqlGetZlVersion = "" +
		"SELECT [objectname],[objecttype],[objectversion],[objectdate] " +
		"FROM [zlversion]"
)

//获取master库连接信息
func GetSQLConfig(server string, sType SType) (*goToolMSSql.MSSqlConfig, error) {
	//CONNECT*AppType*ClientType*ComputerIP*ComputeName
	var chSplit, chEdge byte
	chSplit = 9
	chEdge = 0
	computerIp, err := goToolEnvironment.GetIntranetAddr()
	if err != nil {
		computerIp = "127.0.0.1"
	}
	computerName, err := goToolEnvironment.GetHostName()
	if err != nil {
		computerName = "Test"
	}
	var appType, clientType string
	var port int
	switch sType {
	case ZB:
		appType = "30"
		clientType = "1"
		port = 7050
	case MD:
		appType = "31"
		clientType = "3"
		port = 7051
	default:
		return nil, errors.New(fmt.Sprintf("unexpected type: %s", string(sType)))
	}
	msg := fmt.Sprintf("%cCONNECT%c%s%c%s%c%s%c%s%c",
		chEdge, chSplit, appType, chSplit, clientType, chSplit, computerIp, chSplit, computerName, chEdge)
	r, err := GetSocketMsg(server, port, msg)
	if err != nil {
		return nil, err
	}
	if r == "" {
		return nil, errors.New("socket return empty")
	}

	r = strings.Replace(r, string(chEdge), "", -1)
	connectInfoList := strings.Split(r, string(chSplit))
	if len(connectInfoList) <= 0 {
		return nil, errors.New("socket return msg split empty list")
	}
	if connectInfoList[0] != "RESCONNECT" {
		errMsg := fmt.Sprintf("socket return msg with Prefix %s,exp RESCONNECT", connectInfoList[0])
		return nil, errors.New(errMsg)
	}
	if len(connectInfoList) < 3 {
		errMsg := "socket return msg split without second value and third value"
		return nil, errors.New(errMsg)
	}
	if connectInfoList[1] != "0" && connectInfoList[1] != "1" {
		errMsg := fmt.Sprintf("socket return msg with Prefix %s,exp 0 or 1", connectInfoList[1])
		return nil, errors.New(errMsg)
	}
	if connectInfoList[1] == "0" {
		errMsg := fmt.Sprintf("socket return err msg: %s", connectInfoList[2])
		return nil, errors.New(errMsg)
	}
	if len(connectInfoList) < 5 {
		errMsg := "socket return msg split without sql config"
		return nil, errors.New(errMsg)
	}
	dbConfig := &goToolMSSql.MSSqlConfig{}
	if strings.Index(connectInfoList[2], ",") >= 0 {
		l := strings.Split(connectInfoList[2], ",")
		dbConfig.Server = l[0]
		p, err := strconv.Atoi(l[1])
		if err != nil {
			errMsg := fmt.Sprintf("convert sql port err: %s", err.Error())
			return nil, errors.New(errMsg)
		}
		dbConfig.Port = p
	} else {
		dbConfig.Server = connectInfoList[2]
		dbConfig.Port = 1433
	}
	dbConfig.DbName = "master"
	dbConfig.User = connectInfoList[3]
	dbConfig.Pwd = connectInfoList[4]
	return dbConfig, nil
}

//接收socket消息
func GetSocketMsg(address string, port int, msg string) (string, error) {
	addr := fmt.Sprintf("%s:%d", address, port)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		errMsg := fmt.Sprintf("get tcpAddr err: %s", err.Error())
		return "", errors.New(errMsg)
	}
	conn, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err != nil {
		errMsg := fmt.Sprintf("dialtcp err: %s", err.Error())
		return "", errors.New(errMsg)
	}
	defer func() {
		_ = conn.Close()
	}()
	_, err = conn.Write([]byte(msg))
	if err != nil {
		errMsg := fmt.Sprintf("tcp write data err: %s", err.Error())
		return "", errors.New(errMsg)
	}
	time.Sleep(time.Millisecond * 100)
	_ = conn.CloseWrite()

	result, err := ioutil.ReadAll(conn)
	if err != nil {
		errMsg := fmt.Sprintf("tcp read data err: %s", err.Error())
		return "", errors.New(errMsg)
	}
	result, err = goToolCommon.DecodeGB18030(result)
	if err != nil {
		errMsg := fmt.Sprintf("tcp DecodeGB18030 data err: %s", err.Error())
		return "", errors.New(errMsg)
	}
	return string(result), nil
}

//获取账套列表
func GetAccountList(dbConfig *goToolMSSql2000.MSSqlConfig, sType SType) ([]string, error) {
	var accTable string
	switch sType {
	case ZB:
		accTable = "zlaccount30"
	case MD:
		accTable = "zlraccount31"
	default:
		return nil, errors.New(fmt.Sprintf("unexpected type: %s", string(sType)))
	}
	rows, err := goToolMSSqlHelper.GetRowsBySQL2000(dbConfig, fmt.Sprintf(sqlGetMdAccount, accTable))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	result := make([]string, 0)
	for rows.Next() {
		var acc string
		err := rows.Scan(&acc)
		if err != nil {
			errMsg := fmt.Sprintf("get account list read data err: %s", err.Error())
			return nil, errors.New(errMsg)
		}
		result = append(result, acc)
	}
	if rows.Err() != nil {
		errMsg := fmt.Sprintf("get account list read data err: %s", rows.Err().Error())
		return nil, errors.New(errMsg)
	}
	return result, nil
}

func GetZlCompany(dbConfig *goToolMSSql2000.MSSqlConfig) (*ZlCompany, error) {
	rows, err := goToolMSSqlHelper.GetRowsBySQL2000(dbConfig, sqlGetZlCompany)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	rList := make([]*ZlCompany, 0)
	var coId, coType int
	var coAb, coCode, coUserAb, coUserCode string
	var coAccCrDate time.Time
	for rows.Next() {
		err = rows.Scan(&coId, &coAb, &coCode, &coType, &coUserAb,
			&coUserCode, &coAccCrDate)
		if err != nil {
			return nil, err
		}
		rList = append(rList, &ZlCompany{
			CoId:        coId,
			CoAb:        coAb,
			CoCode:      coCode,
			CoType:      coType,
			CoUserAb:    coUserAb,
			CoUserCode:  coUserCode,
			CoAccCrDate: coAccCrDate,
		})
	}
	if rows.Err() != nil {
		err = rows.Err()
		return nil, err
	}
	if len(rList) > 0 {
		return rList[0], nil
	} else {
		return nil, nil
	}
}

func GetZlVersion(dbConfig *goToolMSSql2000.MSSqlConfig) ([]*ZlVersion, error) {
	rows, err := goToolMSSqlHelper.GetRowsBySQL2000(dbConfig, sqlGetZlVersion)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	rList := make([]*ZlVersion, 0)
	var objectName, objectType, objectVersion string
	var objectDate time.Time
	for rows.Next() {
		err = rows.Scan(&objectName, &objectType, &objectVersion, &objectDate)
		if err != nil {
			return nil, err
		}
		rList = append(rList, &ZlVersion{
			ObjectName:    objectName,
			ObjectType:    objectType,
			ObjectVersion: objectVersion,
			ObjectDate:    objectDate,
		})
	}
	if rows.Err() != nil {
		err = rows.Err()
		return nil, err
	}
	return rList, nil
}
