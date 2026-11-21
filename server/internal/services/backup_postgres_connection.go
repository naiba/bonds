package services

import (
	"net/url"
	"regexp"
	"strings"
)

var postgresKeywordPassword = regexp.MustCompile(`(^|\s)password\s*=\s*('(?:[^'\\]|\\.)*'|[^\s']\S*)\s*`)

// postgresRestoreLockTimeout 限定恢复时等待表锁的时间。恢复要在一个事务里删表重建，
// 需要独占锁：别的会话只要有一个事务一直开着，psql 就会无限期等下去，恢复请求永不返回。
// 超时后整个事务回滚，数据库保持原样。
var postgresRestoreLockTimeout = "60s"

// dropOwnedPublicTablesSQL removes the tables and sequences the connecting
// role owns in the public schema, leaving objects of other owners alone.
const dropOwnedPublicTablesSQL = `DO $$
DECLARE owned record;
BEGIN
  FOR owned IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tableowner = current_user LOOP
    EXECUTE format('DROP TABLE IF EXISTS public.%I CASCADE', owned.tablename);
  END LOOP;
  FOR owned IN SELECT sequencename FROM pg_sequences WHERE schemaname = 'public' AND sequenceowner = current_user LOOP
    EXECUTE format('DROP SEQUENCE IF EXISTS public.%I CASCADE', owned.sequencename);
  END LOOP;
END $$`

// lockOwnedTablesSQL takes the exclusive locks the restore needs up front,
// while lock_timeout is still in force. It has to happen before the dump runs:
// every pg_dump script starts with "SET lock_timeout = 0", so a timeout set
// beforehand no longer applies to the DROP statements inside the dump.
//
// 表不一定在 public 里：数据库或角色的 search_path 可以把它们放进别的 schema。
// 导出文件覆盖整个库，所以这里锁当前角色在所有用户 schema 里拥有的表；只看 public
// 的话，表在别处时一张也锁不到，恢复又会回到无限期等锁。
const lockOwnedTablesSQL = `DO $$
DECLARE owned record;
BEGIN
  FOR owned IN
    SELECT schemaname, tablename FROM pg_tables
    WHERE tableowner = current_user AND schemaname <> 'information_schema' AND schemaname NOT LIKE 'pg\_%'
  LOOP
    EXECUTE format('LOCK TABLE %I.%I IN ACCESS EXCLUSIVE MODE', owned.schemaname, owned.tablename);
  END LOOP;
END $$`

// postgresClientConnection 把 DSN 拆成“可以放在命令行上的连接串”和“只通过环境变量
// 传给 pg_dump / psql 的口令”。命令行参数对同一台机器上的所有用户可见（ps、/proc），
// 以前把完整 DSN 写进 --dbname，备份和恢复期间数据库口令就挂在进程列表里。
// 两种 DSN 写法都要支持：postgres:// URL 和 "host=... password=..." 关键字串。
func postgresClientConnection(dsn string) (connection string, environment []string) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err != nil || parsed.User == nil {
			return dsn, nil
		}
		password, hasPassword := parsed.User.Password()
		if !hasPassword {
			return dsn, nil
		}
		parsed.User = url.User(parsed.User.Username())
		return parsed.String(), []string{"PGPASSWORD=" + password}
	}
	match := postgresKeywordPassword.FindStringSubmatch(dsn)
	if match == nil {
		return dsn, nil
	}
	password := match[2]
	if strings.HasPrefix(password, "'") {
		password = strings.NewReplacer(`\'`, "'", `\\`, `\`).Replace(password[1 : len(password)-1])
	}
	withoutPassword := strings.TrimSpace(postgresKeywordPassword.ReplaceAllString(dsn, "$1"))
	return withoutPassword, []string{"PGPASSWORD=" + password}
}
