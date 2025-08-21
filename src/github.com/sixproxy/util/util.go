package util

import (
	"os"
	"os/exec"
	"strconv"
)

func String2Int(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		return -999999
	}
	return i
}

// getAvailableShell 获取可用的shell
func GetAvailableShell() string {
	shells := []string{"bash", "sh", "/bin/bash", "/bin/sh", "/system/bin/sh"}
	for _, shell := range shells {
		if _, err := exec.LookPath(shell); err == nil {
			return shell
		}
	}
	return ""
}

// newConfigExists 检查新配置是否与备份配置不同
func CheckNewConfigIsSameOldConfig(configPath, backupPath string) bool {
	configData, err1 := os.ReadFile(configPath)
	backupData, err2 := os.ReadFile(backupPath)

	if err1 != nil || err2 != nil {
		return true // 如果无法读取，假设它们不同
	}

	return string(configData) != string(backupData)
}
