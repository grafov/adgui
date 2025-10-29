package commands

import (
	"strings"
	"testing"
)

func TestParseLocationFromStatus(t *testing.T) {
	// Тестовый вывод статуса с ANSI кодами
	testOutput := "Connected to \x1b[1mFRANKFURT\x1b[0m in \x1b[1mTUN\x1b[0m mode, running on \x1b[1mtun0\x1b[0m\n" +
		"Warning: System DNS could not be configured. DNS queries may bypass the VPN tunnel\n" +
		"You can disconnect by running `/opt/adguardvpn_cli/adguardvpn-cli disconnect`\n"

	// Имитируем логику парсинга из checkStatus()
	expectedLocation := "FRANKFURT"

	// Применяем ту же логику, что и в основном коде
	location := testOutput
	prefix := "Connected to "
	if idx := strings.Index(location, prefix); idx >= 0 {
		location = location[idx+len(prefix):]
	}
	// Удаляем ANSI коды
	location = strings.ReplaceAll(location, "\x1b[1m", "")
	location = strings.ReplaceAll(location, "\x1b[0m", "")
	// Удаляем суффикс
	if idx := strings.Index(location, " in "); idx >= 0 {
		location = location[:idx]
	}
	// Очищаем от пробелов
	location = strings.TrimSpace(location)

	if location != expectedLocation {
		t.Errorf("Expected location %q, got %q", expectedLocation, location)
	}
}

func TestParseLocationFromStatusWithDifferentLocation(t *testing.T) {
	// Тестовый вывод статуса с другой локацией
	testOutput := "Connected to \x1b[1mNEW YORK\x1b[0m in \x1b[1mTUN\x1b[0m mode, running on \x1b[1mtun0\x1b[0m\n" +
		"Warning: System DNS could not be configured. DNS queries may bypass the VPN tunnel\n"

	expectedLocation := "NEW YORK"

	// Применяем ту же логику, что и в основном коде
	location := testOutput
	prefix := "Connected to "
	if idx := strings.Index(location, prefix); idx >= 0 {
		location = location[idx+len(prefix):]
	}
	// Удаляем ANSI коды
	location = strings.ReplaceAll(location, "\x1b[1m", "")
	location = strings.ReplaceAll(location, "\x1b[0m", "")
	// Удаляем суффикс
	if idx := strings.Index(location, " in "); idx >= 0 {
		location = location[:idx]
	}
	// Очищаем от пробелов
	location = strings.TrimSpace(location)

	if location != expectedLocation {
		t.Errorf("Expected location %q, got %q", expectedLocation, location)
	}
}
