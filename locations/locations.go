package locations

import (
	"fmt"
	"sort"
	"strings"
)

// Location представляет информацию о локации VPN
type Location struct {
	ISO     string
	Country string
	City    string
	Ping    int
}

// ParseLocations парсит вывод команды list-locations
func ParseLocations(output string) []Location {
	lines := strings.Split(output, "\n")
	var locations []Location

	// Находим позиции колонок по заголовку
	var isoEnd, countryEnd, cityEnd int
	for _, line := range lines {
		cleanLine := strings.ReplaceAll(line, "\x1b[1m", "")
		cleanLine = strings.ReplaceAll(cleanLine, "\x1b[0m", "")
		if strings.Contains(cleanLine, "ISO") && strings.Contains(cleanLine, "COUNTRY") {
			isoEnd = strings.Index(cleanLine, "COUNTRY")
			countryEnd = strings.Index(cleanLine, "CITY")
			cityEnd = strings.Index(cleanLine, "PING")
			break
		}
	}

	if isoEnd == 0 || countryEnd == 0 || cityEnd == 0 {
		fmt.Println("Could not determine column positions from header")
		return locations
	}

	for _, line := range lines {
		// Убираем ANSI escape codes
		cleanLine := strings.ReplaceAll(line, "\x1b[1m", "")
		cleanLine = strings.ReplaceAll(cleanLine, "\x1b[0m", "")

		// Пропускаем пустые строки и заголовок
		if strings.TrimSpace(cleanLine) == "" || strings.Contains(cleanLine, "ISO") {
			continue
		}

		// Извлекаем данные, используя найденные позиции колонок
		var iso, country, city, pingStr string

		// Безопасное извлечение подстрок
		if len(cleanLine) > isoEnd {
			iso = strings.TrimSpace(cleanLine[:isoEnd])
		}
		if len(cleanLine) > countryEnd {
			country = strings.TrimSpace(cleanLine[isoEnd:countryEnd])
		}
		if len(cleanLine) > cityEnd {
			city = strings.TrimSpace(cleanLine[countryEnd:cityEnd])
			pingStr = strings.TrimSpace(cleanLine[cityEnd:])
		} else if len(cleanLine) > countryEnd {
			// Если строка короче, но хватает до countryEnd
			city = strings.TrimSpace(cleanLine[countryEnd:])
			pingStr = ""
		}

		// Пропускаем строки с невалидными данными
		if iso == "" || country == "" || city == "" {
			continue
		}

		// Конвертируем пинг в число
		var ping int
		if pingStr != "" {
			if _, err := fmt.Sscanf(pingStr, "%d", &ping); err != nil {
				// Если не удалось сконвертировать, устанавливаем максимальное значение
				ping = 9999
			}
		} else {
			ping = 9999
		}

		locations = append(locations, Location{
			ISO:     iso,
			Country: country,
			City:    city,
			Ping:    ping,
		})
	}

	// Сортируем локации по пингу (от меньшего к большему)
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].Ping < locations[j].Ping
	})

	return locations
}

// FilterLocations фильтрует локации по имени города или страны
func FilterLocations(locations []Location, query string) []Location {
	if query == "" {
		return locations
	}

	var filtered []Location
	query = strings.ToLower(query)

	for _, loc := range locations {
		if strings.Contains(strings.ToLower(loc.City), query) || strings.Contains(strings.ToLower(loc.Country), query) {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}

// FindFastestLocation находит локацию с минимальным пингом
func FindFastestLocation(locations []Location) *Location {
	if len(locations) == 0 {
		return nil
	}

	fastest := &locations[0]
	for i := 1; i < len(locations); i++ {
		if locations[i].Ping < fastest.Ping {
			fastest = &locations[i]
		}
	}
	return fastest
}
