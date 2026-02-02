// Package ntp — NTP клиент и контроллер (реализация в client_impl.go).
package ntp

// Реконструировано по дизассемблеру: NewController (once.Do+logger), Start→loadConfig, loadConfig→GetStore().GetSources().Range(ConfigureTimeSource), ConfigureTimeSource(key "ntp")→configureAndStartClient/Server, client.offset→(sub1+sub2)>>1, msg.getMode→byte&7.
