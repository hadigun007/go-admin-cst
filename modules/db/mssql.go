// Copyright 2019 GoAdmin Core Team. All rights reserved.
// Use of this source code is governed by a Apache-2.0 style
// license that can be found in the LICENSE file.

package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/GoAdminGroup/go-admin/modules/config"
)

// Mssql is a Connection of mssql.
type Mssql struct {
	Base
}

// GetMssqlDB return the global mssql connection.
func GetMssqlDB() *Mssql {
	return &Mssql{
		Base: Base{
			DbList: make(map[string]*sql.DB),
		},
	}
}

// GetDelimiter implements the method Connection.GetDelimiter.
func (db *Mssql) GetDelimiter() string {
	return "["
}

// GetDelimiter2 implements the method Connection.GetDelimiter2.
func (db *Mssql) GetDelimiter2() string {
	return "]"
}

// GetDelimiters implements the method Connection.GetDelimiters.
func (db *Mssql) GetDelimiters() []string {
	return []string{"[", "]"}
}

// Name implements the method Connection.Name.
func (db *Mssql) Name() string {
	return "mssql"
}

// TODO: ζ΄ηδΌε

func replaceStringFunc(pattern, src string, rpl func(s string) string) (string, error) {

	r, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	bytes := r.ReplaceAllFunc([]byte(src), func(bytes []byte) []byte {
		return []byte(rpl(string(bytes)))
	})

	return string(bytes), nil
}

func replace(pattern string, replace, src []byte) ([]byte, error) {

	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return r.ReplaceAll(src, replace), nil
}

func replaceString(pattern, rep, src string) (string, error) {
	r, e := replace(pattern, []byte(rep), []byte(src))
	return string(r), e
}

func matchAllString(pattern string, src string) ([][]string, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return r.FindAllStringSubmatch(src, -1), nil
}

func isMatch(pattern string, src []byte) bool {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return r.Match(src)
}

func isMatchString(pattern string, src string) bool {
	return isMatch(pattern, []byte(src))
}

func matchString(pattern string, src string) ([]string, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return r.FindStringSubmatch(src), nil
}

// δ»Gfζ‘ζΆε€εΆ
// ε¨ζ§θ‘sqlδΉεε―ΉsqlθΏθ‘θΏδΈζ­₯ε€η
func (db *Mssql) handleSqlBeforeExec(query string) string {
	index := 0
	str, _ := replaceStringFunc("\\?", query, func(s string) string {
		index++
		return fmt.Sprintf("@p%d", index)
	})

	str, _ = replaceString("\"", "", str)

	return db.parseSql(str)
}

//ε°MYSQLηSQLθ―­ζ³θ½¬ζ’δΈΊMSSQLηθ―­ζ³
//1.η±δΊmssqlδΈζ―ζlimitεζ³ζδ»₯ιθ¦ε―ΉmysqlδΈ­ηlimitη¨ζ³εθ½¬ζ’
func (db *Mssql) parseSql(sql string) string {
	//δΈι’ηζ­£εθ‘¨θΎΎεΌεΉιεΊSELECTεINSERTηε³ι?ε­εεε«εδΈεηε€ηοΌε¦ζLIMITεε°LIMITηε³ι?ε­δΉεΉιεΊ
	patten := `^\s*(?i)(SELECT)|(LIMIT\s*(\d+)\s*,\s*(\d+))`
	if !isMatchString(patten, sql) {
		//fmt.Println("not matched..")
		return sql
	}

	res, err := matchAllString(patten, sql)
	if err != nil {
		//fmt.Println("MatchString error.", err)
		return ""
	}

	index := 0
	keyword := strings.TrimSpace(res[index][0])
	keyword = strings.ToUpper(keyword)

	index++
	switch keyword {
	case "SELECT":
		//δΈε«LIMITε³ι?ε­εδΈε€η
		if len(res) < 2 || (!strings.HasPrefix(res[index][0], "LIMIT") && !strings.HasPrefix(res[index][0], "limit")) {
			break
		}

		//δΈε«LIMITεδΈε€η
		if !isMatchString("((?i)SELECT)(.+)((?i)LIMIT)", sql) {
			break
		}

		//ε€ζ­SQLδΈ­ζ―ε¦ε«ζorder by
		selectStr := ""
		orderbyStr := ""
		haveOrderby := isMatchString("((?i)SELECT)(.+)((?i)ORDER BY)", sql)
		if haveOrderby {
			//εorder by ει’ηε­η¬¦δΈ²
			queryExpr, _ := matchString("((?i)SELECT)(.+)((?i)ORDER BY)", sql)

			if len(queryExpr) != 4 || !strings.EqualFold(queryExpr[1], "SELECT") || !strings.EqualFold(queryExpr[3], "ORDER BY") {
				break
			}
			selectStr = queryExpr[2]

			//εorder byθ‘¨θΎΎεΌηεΌ
			orderbyExpr, _ := matchString("((?i)ORDER BY)(.+)((?i)LIMIT)", sql)
			if len(orderbyExpr) != 4 || !strings.EqualFold(orderbyExpr[1], "ORDER BY") || !strings.EqualFold(orderbyExpr[3], "LIMIT") {
				break
			}
			orderbyStr = orderbyExpr[2]
		} else {
			queryExpr, _ := matchString("((?i)SELECT)(.+)((?i)LIMIT)", sql)
			if len(queryExpr) != 4 || !strings.EqualFold(queryExpr[1], "SELECT") || !strings.EqualFold(queryExpr[3], "LIMIT") {
				break
			}
			selectStr = queryExpr[2]
		}

		//εlimitει’ηεεΌθε΄
		first, limit := 0, 0
		for i := 1; i < len(res[index]); i++ {
			if strings.TrimSpace(res[index][i]) == "" {
				continue
			}

			if strings.HasPrefix(res[index][i], "LIMIT") || strings.HasPrefix(res[index][i], "limit") {
				first, _ = strconv.Atoi(res[index][i+1])
				limit, _ = strconv.Atoi(res[index][i+2])
				break
			}
		}

		if haveOrderby {
			sql = fmt.Sprintf("SELECT * FROM (SELECT ROW_NUMBER() OVER (ORDER BY %s) as ROWNUMBER_, %s   ) as TMP_ WHERE TMP_.ROWNUMBER_ > %d AND TMP_.ROWNUMBER_ <= %d", orderbyStr, selectStr, first, limit)
		} else {
			if first == 0 {
				first = limit
			} else {
				first = limit - first
			}
			sql = fmt.Sprintf("SELECT * FROM (SELECT TOP %d * FROM (SELECT TOP %d %s) as TMP1_ ) as TMP2_ ", first, limit, selectStr)
		}
	default:
	}
	return sql
}

// QueryWithConnection implements the method Connection.QueryWithConnection.
func (db *Mssql) QueryWithConnection(con string, query string, args ...interface{}) ([]map[string]interface{}, error) {
	query = db.handleSqlBeforeExec(query)
	return CommonQuery(db.DbList[con], query, args...)
}

// ExecWithConnection implements the method Connection.ExecWithConnection.
func (db *Mssql) ExecWithConnection(con string, query string, args ...interface{}) (sql.Result, error) {
	query = db.handleSqlBeforeExec(query)
	return CommonExec(db.DbList[con], query, args...)
}

// Query implements the method Connection.Query.
func (db *Mssql) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	query = db.handleSqlBeforeExec(query)
	return CommonQuery(db.DbList["default"], query, args...)
}

// Exec implements the method Connection.Exec.
func (db *Mssql) Exec(query string, args ...interface{}) (sql.Result, error) {
	query = db.handleSqlBeforeExec(query)
	return CommonExec(db.DbList["default"], query, args...)
}

func (db *Mssql) QueryWith(tx *sql.Tx, conn, query string, args ...interface{}) ([]map[string]interface{}, error) {
	if tx != nil {
		return db.QueryWithTx(tx, query, args...)
	}
	return db.QueryWithConnection(conn, query, args...)
}

func (db *Mssql) ExecWith(tx *sql.Tx, conn, query string, args ...interface{}) (sql.Result, error) {
	if tx != nil {
		return db.ExecWithTx(tx, query, args...)
	}
	return db.ExecWithConnection(conn, query, args...)
}

// InitDB implements the method Connection.InitDB.
func (db *Mssql) InitDB(cfgs map[string]config.Database) Connection {
	db.Configs = cfgs
	db.Once.Do(func() {
		for conn, cfg := range cfgs {

			sqlDB, err := sql.Open("sqlserver", cfg.GetDSN())

			if sqlDB == nil {
				panic("invalid connection")
			}

			if err != nil {
				_ = sqlDB.Close()
				panic(err.Error())
			}

			sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
			sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
			sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
			sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

			db.DbList[conn] = sqlDB

			if err := sqlDB.Ping(); err != nil {
				panic(err)
			}
		}
	})
	return db
}

// BeginTxWithReadUncommitted starts a transaction with level LevelReadUncommitted.
func (db *Mssql) BeginTxWithReadUncommitted() *sql.Tx {
	return CommonBeginTxWithLevel(db.DbList["default"], sql.LevelReadUncommitted)
}

// BeginTxWithReadCommitted starts a transaction with level LevelReadCommitted.
func (db *Mssql) BeginTxWithReadCommitted() *sql.Tx {
	return CommonBeginTxWithLevel(db.DbList["default"], sql.LevelReadCommitted)
}

// BeginTxWithRepeatableRead starts a transaction with level LevelRepeatableRead.
func (db *Mssql) BeginTxWithRepeatableRead() *sql.Tx {
	return CommonBeginTxWithLevel(db.DbList["default"], sql.LevelRepeatableRead)
}

// BeginTx starts a transaction with level LevelDefault.
func (db *Mssql) BeginTx() *sql.Tx {
	return CommonBeginTxWithLevel(db.DbList["default"], sql.LevelDefault)
}

// BeginTxWithLevel starts a transaction with given transaction isolation level.
func (db *Mssql) BeginTxWithLevel(level sql.IsolationLevel) *sql.Tx {
	return CommonBeginTxWithLevel(db.DbList["default"], level)
}

// BeginTxWithReadUncommittedAndConnection starts a transaction with level LevelReadUncommitted and connection.
func (db *Mssql) BeginTxWithReadUncommittedAndConnection(conn string) *sql.Tx {
	return CommonBeginTxWithLevel(db.DbList[conn], sql.LevelReadUncommitted)
}

// BeginTxWithReadCommittedAndConnection starts a transaction with level LevelReadCommitted and connection.
func (db *Mssql) BeginTxWithReadCommittedAndConnection(conn string) *sql.Tx {
	return CommonBeginTxWithLevel(db.DbList[conn], sql.LevelReadCommitted)
}

// BeginTxWithRepeatableReadAndConnection starts a transaction with level LevelRepeatableRead and connection.
func (db *Mssql) BeginTxWithRepeatableReadAndConnection(conn string) *sql.Tx {
	return CommonBeginTxWithLevel(db.DbList[conn], sql.LevelRepeatableRead)
}

// BeginTxAndConnection starts a transaction with level LevelDefault and connection.
func (db *Mssql) BeginTxAndConnection(conn string) *sql.Tx {
	return CommonBeginTxWithLevel(db.DbList[conn], sql.LevelDefault)
}

// BeginTxWithLevelAndConnection starts a transaction with given transaction isolation level and connection.
func (db *Mssql) BeginTxWithLevelAndConnection(conn string, level sql.IsolationLevel) *sql.Tx {
	return CommonBeginTxWithLevel(db.DbList[conn], level)
}

// QueryWithTx is query method within the transaction.
func (db *Mssql) QueryWithTx(tx *sql.Tx, query string, args ...interface{}) ([]map[string]interface{}, error) {
	query = db.handleSqlBeforeExec(query)
	return CommonQueryWithTx(tx, query, args...)
}

// ExecWithTx is exec method within the transaction.
func (db *Mssql) ExecWithTx(tx *sql.Tx, query string, args ...interface{}) (sql.Result, error) {
	query = db.handleSqlBeforeExec(query)
	return CommonExecWithTx(tx, query, args...)
}
