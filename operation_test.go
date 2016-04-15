package msops

import (
	"strconv"
	"testing"
)

func TestReadDataSet(t *testing.T) {
	data, err := readDataSet(testEndpoint1, "SELECT * from data_test.tbl_test where id = ?", 10000)
	if err != nil {
		t.Errorf("Test readDataSet error: %s", err.Error())
	} else if len(data) != 1 {
		t.Errorf("Test readDataSet failed: actual rowcount %d, expected 2", len(data))
	} else {
		row := data[0]
		if len(row) != 2 {
			t.Errorf("Test readDataSet failed: actual colcount %d, expected 2", len(row))
		} else if val, exist := row["name"]; !exist {
			t.Error("Test readDataSet failed: col 'name' is not found")
		} else if actData := getString(val); actData != "world" {
			t.Errorf("Test readDataSet failed: actual value of col 'name' is %s, expected hello", actData)
		} else if val, exist = row["id"]; !exist {
			t.Error("Test readDataSet failed: col 'id' is not found")
		} else if actKey := getInt(val); actKey != 10000 {
			t.Errorf("Test readDataSet failed: actual value of col 'id' is %d, expected 1", actKey)
		}
	}
	if _, err = readDataSet(badEndpoint, "SELECT * from data_test.tbl_test where id = ?", 10000); err == nil {
		t.Error("BadEndpoint readDataSet should cause error")
	}
}

func TestGetMasterStatus(t *testing.T) {
	if masterSt, err := GetMasterStatus(testEndpoint1); err != nil {
		t.Errorf("Test GetMasterStatus error: %s", err.Error())
	} else if masterSt.File != "binlog.000001" {
		t.Errorf("Test GetMasterStatus failed: actual master log file %s, expected binlog.000001", masterSt.File)
	}

	if _, err := GetMasterStatus(badEndpoint); err == nil {
		t.Error("Get badEndpoint master status should cause error")
	}
}

func TestGetGlobalVariables(t *testing.T) {
	if portMap, err := GetGlobalVariables(testEndpoint1, "port"); err != nil {
		t.Errorf("Test GetGlobalVariables error: %s", err.Error())
	} else if val, exist := portMap["port"]; !exist {
		t.Errorf("Test GetGlobalVariables failed: port is not existed")
	} else if val != "3306" {
		t.Errorf("Test GetGlobalVariables failed: actual port is %s, expected 3306", val)
	}

	if portMap, err := GetGlobalVariables(testEndpoint2, "%%server%"); err != nil {
		t.Errorf("Test GetGlobalVariables error: %s", err.Error())
	} else if val, exist := portMap["server_id"]; !exist {
		t.Errorf("Test GetGlobalVariables failed: server_id is not existed")
	} else if val != "2" {
		t.Errorf("Test GetGlobalVariables failed: actual port is %s, expected 2", val)
	}

	if _, err := GetMasterStatus(badEndpoint); err == nil {
		t.Error("Get badEndpoint global variables status should cause error")
	}
}

func TestSetGlobalVariable(t *testing.T) {
	expireLogsDays := 50
	if err := SetGlobalVariable(testEndpoint1, "expire_logs_days", expireLogsDays); err != nil {
		t.Errorf("Test SetGlobalVariable error: %s", err.Error())
	} else if res, err := GetGlobalVariables(testEndpoint1, "expire_logs_days"); err != nil {
		t.Errorf("Test SetGlobalVariable error: %s", err.Error())
	} else if res["expire_logs_days"] != strconv.Itoa(expireLogsDays) {
		t.Errorf("Test SetGlobalVariable failed: actual expire_logs_days is %s, expected 100", res["expire_logs_days"])
	}

	if SetGlobalVariable(badEndpoint, "expire_logs_days", expireLogsDays) == nil {
		t.Error("Set badEndpoint global variables should cause error")
	}

	if err := SetGlobalVariable(unregisteredEndpoint, "expire_logs_days", expireLogsDays); err != errNotRegistered {
		t.Error("Set unregisteredEndpoint global variables should throw errNotRegistered")
	}

	if err := SetGlobalVariable(testEndpoint1, ";drop mysql", expireLogsDays); err != errKeyInvalid {
		t.Error("Set global variables with invalid key should throw errKeyInvalid")
	}
}

func TestGetInndoDBStatus(t *testing.T) {
	if _, err := GetInnoDBStatus(testEndpoint1); err != nil {
		t.Errorf("Test GetInndoDBStatus error: %s", err.Error())
	}

	if _, err := GetInnoDBStatus(badEndpoint); err == nil {
		t.Error("Get badEndpoint innodb engine status should cause error")
	}
}
