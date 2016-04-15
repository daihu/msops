package msops

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
)

// Instance records the connect information.
type Instance struct {
	dbaUser       string
	dbaPassword   string
	replUser      string
	replPassword  string
	connectParams map[string]string
	connection    *sql.DB
}

// ReplicationStatus represents the replication status between to instance.
//
// The judgement is according to the result of `SHOW SLAVE STATUS` and `SHOW MASTER STATUS`.
//
// Comparing the binlog file and binlog positions between master and slave.
type ReplicationStatus int

// InstanceStatus represents the running status of one instance.
//
// The judgement is according to the result of db.Conn.Ping().
type InstanceStatus int

const (
	// ReplStatusOK implies that in the slave status of the slave instance,
	// 'Master_Host' and 'Master_Port' are the same as the master's,
	// 'Slave_SQL_Running' and 'Slave_IO_Running' are both 'Yes',
	// 'Master_Log_File' and 'Master_Log_Position'equals to '0'.
	ReplStatusOK ReplicationStatus = iota

	// ReplStatusError implies that in the slave status of the slave instance,
	// 'Master_Host' and 'Master_Port' are the same as the master's,
	// 'Slave_SQL_Running' and 'Slave_IO_Running' are not both 'Yes',
	// and 'Last_Error' is not empty.
	ReplStatusError

	// ReplStatusSyning implies that in the slave status of the slave instance,
	// 'Master_Host' and 'Master_Port' are the same as the master's,
	// 'Slave_SQL_Running' and 'Slave_IO_Running' are both 'Yes',
	// 'Second_Behind_Master' is larger than '0'.
	ReplStatusSyning

	// ReplStatusPausing implies that in the slave status of the slave instance,
	// 'Master_Host' and 'Master_Port' are the same as the master's,
	// and 'Slave_SQL_Running' and 'Slave_IO_Running' are both 'no'.
	ReplStatusPausing

	// ReplStatusWrongMaster implies that in the slave status of the slave instance,
	// 'Master_Host' and 'Master_Port' are not the same as the master's.
	ReplStatusWrongMaster

	// ReplStatusNone implies that the slave status of the endpoint is empty.
	ReplStatusNone

	// ReplStatusUnknown implies that we can't connect to the slave instance.
	ReplStatusUnknown
)

const (
	// InstanceOK implies that we can connect to the instance.
	InstanceOK InstanceStatus = iota

	// InstanceERROR implies that we can't connect to the instance.
	InstanceERROR

	// InstanceUnregistered implies that we haven't registered the instance.
	InstanceUnregistered
)

const driverName = "mysql"

var (
	connectionPool   = make(map[string]*Instance)
	errNotRegistered = errors.New("the instance is not registered")
	emptySlaveStatus = SlaveStatus{}
)

// Register registers the instance of endpoint with opening the connection with user 'dbaUser', password 'dbaPassword'.
//
// 'replUser' and 'replPassword' are used to be established replication by other endpoints.
//
// 'params' are the k-v params appending to go-mysql-driver connection string.
//
// 'dbaUser' should have the following privileges at least: RELOAD, PROCESS, SUPER, REPLICATION CLIENT, REPLICATION SLAVE.
//
// 'replUser' should have the following privileges at least: PROCESS, REPLICATION SLAVE.
//
// 'endpoint' show have the form "host:port".
//
// If the final connection string generated is invalid, an error will be returned.
func Register(endpoint, dbaUser, dbaPassword, replUser, replPassword string, params map[string]string) error {
	if _, exist := connectionPool[endpoint]; !exist {
		if params == nil {
			params = make(map[string]string)
		}
		params["interpolateParams"] = "true"
		paramSlice := make([]string, 0, len(params))
		for key, value := range params {
			paramSlice = append(paramSlice, fmt.Sprintf("%s=%s", key, value))
		}
		connStr := fmt.Sprintf("%s:%s@tcp(%s)/?%s", dbaUser, dbaPassword, endpoint, strings.Join(paramSlice, "&"))
		var conn *sql.DB
		var err error
		if conn, err = sql.Open(driverName, connStr); err != nil {
			return err
		}
		inst := &Instance{
			dbaUser:       dbaUser,
			dbaPassword:   dbaPassword,
			replUser:      replUser,
			replPassword:  replPassword,
			connectParams: params,
			connection:    conn,
		}
		connectionPool[endpoint] = inst
	}
	return nil
}

// Unregister deletes the information from msops's connection pool and close the connections to endpoint.
func Unregister(endpoint string) {
	if inst, exist := connectionPool[endpoint]; exist {
		inst.connection.Close()
	}
	delete(connectionPool, endpoint)
}

// CheckInstance checks the status of a instance with the endpoint.
func CheckInstance(endpoint string) InstanceStatus {
	if inst, exist := connectionPool[endpoint]; exist {
		if inst.connection.Ping() == nil {
			return InstanceOK
		}
		return InstanceERROR
	}
	return InstanceUnregistered
}

// CheckReplication checks the replicaton status between slaveEndpoint and masterEndpoint.
// Note that if one of slave or master is not registered,
// or getting MasterStatus and SlaveStatus failed, ReplStatusUnknown is returned.
func CheckReplication(slaveEndpoint, masterEndpoint string) ReplicationStatus {
	if CheckInstance(slaveEndpoint) == InstanceUnregistered ||
		CheckInstance(masterEndpoint) == InstanceUnregistered {
		return ReplStatusUnknown
	}
	var masterStatus MasterStatus
	var slaveStatus SlaveStatus
	var err error
	if masterStatus, err = GetMasterStatus(masterEndpoint); err != nil {
		return ReplStatusUnknown
	}
	if slaveStatus, err = GetSlaveStatus(slaveEndpoint); err != nil {
		return ReplStatusUnknown
	}
	if reflect.DeepEqual(emptySlaveStatus, slaveStatus) {
		return ReplStatusNone
	}
	if net.JoinHostPort(slaveStatus.MasterHost, strconv.Itoa(slaveStatus.MasterPort)) != masterEndpoint {
		return ReplStatusWrongMaster
	}
	if slaveStatus.LastErrno != 0 {
		return ReplStatusError
	}
	if slaveStatus.SlaveSQLRunning == "No" && slaveStatus.SlaveIORunning == "No" {
		return ReplStatusPausing
	}
	if slaveStatus.MasterLogFile != masterStatus.File ||
		slaveStatus.ExecMasterLogPos != masterStatus.Position {
		return ReplStatusSyning
	}
	return ReplStatusOK
}
