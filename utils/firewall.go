package utils

import (
	"log"
	"regexp"
)

// FirewallRule 防火墙规则类型
type FirewallRule struct {
	ID      int
	Type    string // "whitelist" 或 "blacklist"
	Pattern string // 正则表达式模式
	Active  bool
}

// InitFirewall 初始化防火墙模块
func InitFirewall() {
	// 创建防火墙规则表
	createFirewallTable()
}

// createFirewallTable 创建防火墙规则表
func createFirewallTable() {
	db := GetDB()
	query := `
	CREATE TABLE IF NOT EXISTS firewall_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL, -- 'whitelist' 或 'blacklist'
		pattern TEXT NOT NULL,
		active BOOLEAN NOT NULL DEFAULT 1
	);
	`
	
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Failed to create firewall_rules table: %v", err)
	}
}

// AddFirewallRule 添加防火墙规则
// 参数:
//   ruleType - 规则类型("whitelist"或"blacklist")
//   pattern - 正则表达式模式
// 返回: error - 添加过程中的错误
func AddFirewallRule(ruleType, pattern string) error {
	db := GetDB()
	query := `INSERT INTO firewall_rules (type, pattern, active) VALUES (?, ?, ?)`
	_, err := db.Exec(query, ruleType, pattern, true)
	return err
}

// GetFirewallRules 获取所有防火墙规则
// 返回: 
//   []*FirewallRule - 防火墙规则列表
//   error - 查询过程中的错误
func GetFirewallRules() ([]*FirewallRule, error) {
	db := GetDB()
	query := `SELECT id, type, pattern, active FROM firewall_rules WHERE active = 1 ORDER BY id`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var rules []*FirewallRule
	for rows.Next() {
		var rule FirewallRule
		err := rows.Scan(&rule.ID, &rule.Type, &rule.Pattern, &rule.Active)
		if err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}
	
	return rules, nil
}

// DeleteFirewallRule 删除防火墙规则
// 参数: id - 规则ID
// 返回: error - 删除过程中的错误
func DeleteFirewallRule(id int) error {
	db := GetDB()
	query := `DELETE FROM firewall_rules WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

// IsAddressAllowed 检查目标地址是否被允许
// 参数: address - 目标地址
// 返回: bool - 是否允许连接
func IsAddressAllowed(address string) bool {
	rules, err := GetFirewallRules()
	if err != nil {
		log.Printf("Failed to get firewall rules: %v", err)
		// 出错时默认允许连接
		return true
	}
	
	// 如果没有规则，默认允许所有流量
	if len(rules) == 0 {
		return true
	}
	
	// 检查白名单规则
	whitelistExists := false
	for _, rule := range rules {
		if rule.Type == "whitelist" {
			whitelistExists = true
			match, err := regexp.MatchString(rule.Pattern, address)
			if err != nil {
				log.Printf("Invalid whitelist pattern '%s': %v", rule.Pattern, err)
				continue
			}
			if match {
				return true
			}
		}
	}
	
	// 如果存在白名单但地址不匹配任何白名单规则，则拒绝
	if whitelistExists {
		return false
	}
	
	// 检查黑名单规则
	for _, rule := range rules {
		if rule.Type == "blacklist" {
			match, err := regexp.MatchString(rule.Pattern, address)
			if err != nil {
				log.Printf("Invalid blacklist pattern '%s': %v", rule.Pattern, err)
				continue
			}
			// 如果匹配黑名单规则，则拒绝
			if match {
				return false
			}
		}
	}
	
	// 如果没有匹配任何黑名单规则，则允许
	return true
}