package usecase

import (
	"time"

	"puasa-sunnah/internal/domains/fasting"
)

func text(language fasting.Language, english, indonesian string) string {
	if language == fasting.LanguageIndonesian {
		return indonesian
	}
	return english
}

func weekdayName(language fasting.Language, weekday time.Weekday) string {
	if language != fasting.LanguageIndonesian {
		return weekday.String()
	}

	return [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}[weekday]
}

func disclaimer(language fasting.Language) string {
	return text(language,
		"Planning information only. Hijri dates can differ by location and authority; verify the accepted date with your local Islamic authority. This API does not issue a fatwa.",
		"Informasi ini hanya untuk perencanaan. Tanggal Hijriah dapat berbeda menurut lokasi dan otoritas; pastikan tanggal yang diterima oleh otoritas Islam setempat. API ini tidak mengeluarkan fatwa.",
	)
}

func calendarCaveat(language fasting.Language, calendar fasting.Calendar) string {
	if calendar == fasting.CalendarUmmAlQura {
		return text(language,
			"Umm al-Qura is Saudi Arabia's civil planning calendar. Its dates can differ from calendars accepted by other authorities and localities; verify dates used for worship with the relevant authority.",
			"Umm al-Qura adalah kalender sipil untuk perencanaan di Arab Saudi. Tanggalnya dapat berbeda dari kalender yang diterima otoritas dan wilayah lain; pastikan tanggal ibadah kepada otoritas yang relevan.",
		)
	}

	return text(language,
		"For worship, Ramadan, Shawwal, and Dhu al-Hijjah dates can be superseded by an official Kemenag Sidang Isbat decision.",
		"Untuk keperluan ibadah, tanggal Ramadan, Syawal, dan Zulhijah dapat berubah mengikuti keputusan resmi Sidang Isbat Kemenag.",
	)
}

func rulesIndonesian() []fasting.Rule {
	return []fasting.Rule{
		{Key: "monday_thursday", Name: "Puasa Senin dan Kamis", Kind: "recommended", Calculation: "Hari dalam kalender Masehi adalah Senin atau Kamis.", Evidence: []string{"https://sunnah.com/tirmidhi:747", "https://sunnah.com/muslim:1162b"}},
		{Key: "white_days", Name: "Ayyamul Bidh", Kind: "recommended", Calculation: "Tanggal 13, 14, atau 15 setiap bulan Hijriah, kecuali terdapat larangan dengan prioritas lebih tinggi.", Evidence: []string{"https://sunnah.com/nasai:2420", "https://sunnah.com/abudawud:2449"}},
		{Key: "six_shawwal_candidate", Name: "Enam hari puasa Syawal", Kind: "recommended", Calculation: "Enam hari mana pun sejak 2 Syawal sampai akhir bulan; pengguna perlu mencatat sendiri hari yang telah dijalankan.", Evidence: []string{"https://sunnah.com/muslim:1164a"}, Caveat: "Terdapat perbedaan pendapat mengenai rincian seperti pelaksanaan berturut-turut dan penggabungan niat."},
		{Key: "ashura", Name: "Asyura dengan hari pendamping", Kind: "recommended", Calculation: "10 Muharam, sebaiknya disertai 9 Muharam; 11 Muharam lazim disebut ulama sebagai alternatif.", Evidence: []string{"https://sunnah.com/muslim:1134b", "https://sunnah.com/muslim:1162b"}, Caveat: "Pengklasifikasi hanya menampilkan 9 dan 10 Muharam karena teks primer yang dirujuk secara eksplisit menyebut penambahan hari kesembilan."},
		{Key: "first_nine_dhul_hijjah", Name: "Sembilan hari pertama Zulhijah", Kind: "recommended", Calculation: "Bulan Hijriah ke-12, tanggal 1–9; tanggal 9 adalah Arafah.", Evidence: []string{"https://sunnah.com/abudawud:2437"}},
		{Key: "arafah", Name: "Puasa Arafah", Kind: "recommended", Calculation: "9 Zulhijah bagi yang tidak berhaji.", Evidence: []string{"https://sunnah.com/muslim:1162b"}, Caveat: "Anjuran ini tidak berlaku dengan cara yang sama bagi jemaah haji yang sedang wukuf di Arafah."},
		{Key: "shaban", Name: "Memperbanyak puasa pada Syakban", Kind: "recommended_pattern", Calculation: "Pola berpuasa pada sebagian besar bulan Hijriah ke-8, bukan aturan pada tanggal tertentu.", Evidence: []string{"https://sunnah.com/bukhari:1969"}, Caveat: "Tidak ditampilkan oleh pengklasifikasi tanggal karena merupakan pola pribadi dan memiliki rincian fikih tambahan menjelang Ramadan."},
		{Key: "three_monthly", Name: "Tiga hari setiap bulan", Kind: "recommended_pattern", Calculation: "Tiga hari mana pun setiap bulan Hijriah; tanggal 13–15 dikenali secara terpisah sebagai Ayyamul Bidh.", Evidence: []string{"https://sunnah.com/bukhari:1981", "https://sunnah.com/nasai:2420"}},
		{Key: "dawud", Name: "Puasa Nabi Dawud", Kind: "recommended_pattern", Calculation: "Berselang-seling antara satu hari berpuasa dan satu hari tidak berpuasa, dengan tetap menghindari hari yang dilarang.", Evidence: []string{"https://sunnah.com/bukhari:1976"}, Caveat: "Memerlukan riwayat pribadi sehingga tidak ditampilkan oleh pengklasifikasi tanggal yang stateless."},
		{Key: "eid", Name: "Dua hari Id", Kind: "prohibited", Calculation: "1 Syawal dan 10 Zulhijah.", Evidence: []string{"https://sunnah.com/bukhari:1990"}},
		{Key: "tashriq", Name: "Hari Tasyrik", Kind: "prohibited", Calculation: "11, 12, dan 13 Zulhijah.", Evidence: []string{"https://sunnah.com/muslim:1141a"}, Caveat: "Terdapat pengecualian terbatas dalam nash bagi jemaah haji tertentu yang tidak dapat mempersembahkan hewan kurban."},
		{Key: "friday_alone", Name: "Puasa Jumat secara tersendiri", Kind: "conditional_caution", Calculation: "Memerlukan informasi mengenai puasa pada hari sebelum atau sesudahnya dan apakah terdapat puasa lain yang disyariatkan pada hari Jumat.", Evidence: []string{"https://sunnah.com/bukhari:1985"}, Caveat: "Tidak ditampilkan oleh pengklasifikasi stateless; tanyakan keadaan pribadi kepada ulama yang kompeten."},
	}
}
