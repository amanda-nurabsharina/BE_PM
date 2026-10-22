# IMPLEMENTATION PLAN V2: CREATORTASK
### Platform Manajemen Kampanye (Endorse & Barter), Invoice Generator, Buku Kas & Pelaporan PDF

> **Dokumen Rujukan:** Software Requirements Specification (SRS) CreatorTask v1.0 & Mockup Web View  
> **Arsitektur Referensi:**  
> - **Backend:** Go 1.25+ / GoFiber v2 / GORM / SQLite & PostgreSQL (*mengikuti standar `BE_COMPRO`*)  
> - **Frontend:** Next.js 16+ App Router / React 19 / TypeScript / Tailwind CSS v4 / Lucide React (*mengikuti standar `FE_COMPRO`*)  
> **Target Direktori Proyek:** `BE_PM` (Backend API) dan `FE_PM` (Frontend Web App)

---

## 1. Ringkasan Eksekutif & Resolusi Kesenjangan Spesifikasi (SRS Bab 10)

Berdasarkan telaah mendalam terhadap **SRS CreatorTask (Halaman 1–40)**, dokumen rencana implementasi sebelumnya (`implementation_plan.md`), serta tangkapan layar **Mockup Web View**, dokumen ini menyusun arsitektur sistem baru yang production-ready dengan standar keamanan tinggi (*enterprise-grade code security*).

### Resolusi 14 Temuan Celah Bisnis (SRS Bab 10):
1. **Autentikasi & Keamanan Sesi**: Menggantikan validasi lokal tanpa verifikasi dengan backend JWT Auth (`pkg/security` Bcrypt + JWT token). Sesi akun cepat (*Quick Switcher*) tetap disediakan untuk kemudahan demo/evaluasi dengan memanggil endpoint `/api/v1/auth/quick-login`. Token menggunakan rotasi kedaluwarsa yang aman (bukan permanent 100 tahun) serta rate limiter khusus brute-force.
2. **Filter Periode Bulan Dinamis**: Dropdown bulan dan tahun tidak lagi di-hardcode, melainkan di-generate secara dinamis oleh backend dari rentang tanggal data riil (`dueDate` tugas, `date` transaksi, `issueDate` invoice) plus bulan berjalan.
3. **Proteksi Status Mundur & Paid Lock**: Status tugas `paid` (Sudah Dibayar) dan invoice `paid` **dikunci secara permanen**. Tidak dapat diturunkan statusnya menjadi `draft` atau `waiting_payment` guna mencegah korupsi data kas.
4. **Proteksi Integritas Hapus Terkait Kas & Invoice**: 
   - Tugas dengan status masih `draft` atau `waiting_payment` dapat dihapus (dan invoice draft terkait otomatis dilepas/dibersihkan).
   - Tugas yang sudah `paid` **diblokir dari penghapusan** (tombol hapus dinonaktifkan/disembunyikan).
   - Transaksi kas hasil pelunasan invoice (`relatedInvoiceId` tidak null) **diblokir mutlak dari penghapusan di halaman Keuangan** (BR-09).
5. **Integritas Pajak (PPN)**: Ketika invoice telah `paid`, sakelar PPN dikunci. Subtotal, PPN, dan Grand Total yang tercatat menjadi permanen untuk kepatuhan pembukuan fiskal.
6. **Pemisahan Valuasi Barter vs Kas Nyata**: Pada modul Laporan dan KPI, nilai proyek Barter disajikan tersendiri sebagai *Valuasi Produk Barter (Non-Tunai)* sehingga tidak menggelembungkan piutang kas tunai (*Belum Lunas*).
7. **Katalog Audit Log Standar**: Mengadopsi katalog lengkap Lampiran A SRS (10 kode aksi standar). Tabel Audit Log bersifat **Immutable (Append-Only)** tanpa endpoint edit/hapus.
8. **Format & Penomoran Invoice Unik**: Menggunakan format otomatis `INV/{TYPE}/{YEAR}/{SEQUENCE}` (contoh: `INV/ENDORSE/2026/001`, `INV/AFFILIATE/2026/002`), dijamin unik oleh unique constraint database dan transaksi atomik dengan table-locking sequence counter.
9. **Penegasan KPI Hero Beranda**: Memisahkan metrik *Saldo Kas Riil Saat Ini* (seluruh kas masuk dikurangi kas keluar) dengan *Piutang Pending Bulan Berjalan*.
10. **Akurasi Nilai Uang (Anti Floating-Point Drift)**: Mengubah seluruh nominal moneter dari `float64` menjadi **`int64` (Rupiah murni)**. Menghilangkan risiko galat floating-point IEEE-754 pada kalkulasi PPN dan rekonsiliasi kas.
11. **Penyimpanan Terpusat & Sinkronisasi API**: Seluruh state disimpan di database melalui REST API backend. Tersedia endpoint ekspor JSON/Backup untuk keamanan data.
12. **Validasi & Kompresi Media Tingkat Lanjut**: Endpoint `/api/v1/upload` membatasi berkas maksimal 2 MB di tingkat engine Fiber (`BodyLimit`), memverifikasi **Magic Bytes** file (`http.DetectContentType`), menolak SVG (anti-XSS), dan menyimpan berkas terkompresi di `./uploads/` dengan UUID.
13. **Konsistensi Tipe Data Filter**: Enum filter diselaraskan: `endorse`, `barter`, `other`. Opsi gabungan `Endorse + Barter` adalah filter query agregasi, bukan entitas baru.
14. **Deteksi Status Terlambat (Overdue)**: Sistem secara otomatis menandai invoice atau tugas yang melewati `dueDate` dengan indikator visual *Overdue* di Beranda dan Invoice.

---

## 2. Arsitektur Backend & Keamanan Kode (`BE_PM` - Standar `BE_COMPRO` Diperketat)

### 2.1 Standar Keamanan Kode Utama (Security Standards)
1. **Pencegahan IDOR (Insecure Direct Object References)**: Setiap query dan mutasi database (`SELECT`, `UPDATE`, `DELETE`) **wajib menyertakan parameter `user_id = ?`** yang diambil langsung dari token JWT terverifikasi (`c.Locals("user")`). Tidak ada endpoint yang mempercayai ID entitas tanpa verifikasi kepemilikan user.
2. **Anti Race-Condition pada Pelunasan**: Menambahkan constraint `UNIQUE(related_invoice_id)` pada tabel `transactions` dan pessimistic locking `tx.Clauses(clause.Locking{Strength: "UPDATE"})` saat pelunasan. Hal ini secara fisik mencegah duplikasi pencatatan kas jika tombol pelunasan diklik dua kali secara simultan.
3. **Pemberantasan Floating Point Bug (IDR Integer Standard)**: Seluruh field moneter (`fee_or_value`, `subtotal`, `tax_amount`, `grand_total`, `amount`) menggunakan tipe **`int64`**. IDR tidak mengenal pecahan desimal, sehingga `int64` menjamin kalkulasi 100% presisi.
4. **Validasi Berkas Multi-Lapis (Secure File Upload)**:
   - Body size limit: 2 MB via `fiber.Config{BodyLimit: 2 * 1024 * 1024}`.
   - Header extension whitelist: `.jpg`, `.jpeg`, `.png`, `.webp`, `.pdf`.
   - **Content-Type Magic Bytes Check**: Membaca 512 byte pertama dengan `http.DetectContentType(buffer)` untuk memastikan isi berkas asli dan bukan script/executable yang disamarkan.
   - Penolakan berkas SVG untuk mencegah Stored XSS.
   - Header penyajian berkas statis: `X-Content-Type-Options: nosniff`.
5. **Anti Brute-Force pada Login**: Rate limiter ketat pada rute `/api/v1/auth/login` (maksimal 5 percobaan per menit per alamat IP).
6. **Token Expiration Realistis**: Mengganti token 100 tahun menjadi Access Token 24 Jam dengan opsi perpanjangan sesi yang aman.
7. **Penyaringan Input (DTO Validation & Anti-XSS)**: Menggunakan `go-playground/validator/v10` di backend dan validasi schema `Zod` di frontend. Kanvas TTD divalidasi ketat hanya menerima format `data:image/png;base64,...`.

### 2.1 Struktur Direktori Backend
```
BE_PM/
├── cmd/
│   └── server/
│       └── main.go                 # Bootstrapping Fiber, middleware global, routing
├── config/
│   └── config.go               # Environment loader (.env), DB connection strings, JWT secret
├── internal/
│   ├── domain/
│   │   └── entity.go           # GORM Structs: User, Task, ChecklistItem, Invoice, Item, Transaction, AuditLog
│   ├── dto/
│   │   └── dto.go              # Request & Response Data Transfer Objects dengan validasi
│   ├── handler/
│   │   ├── auth_handler.go     # Login, Quick Login, Profil
│   │   ├── task_handler.go     # CRUD Tugas, checklist toggle, pelunasan cascade
│   │   ├── invoice_handler.go  # Invoice CRUD, toggle PPN, tandai lunas
│   │   ├── cashflow_handler.go # Buku kas transaksi, filter, ringkasan saldo
│   │   ├── report_handler.go   # Agregasi laporan bulanan & data cetak PDF
│   │   ├── setting_handler.go  # Konfigurasi studio, profil, PPN, TTD, bank
│   │   ├── audit_handler.go    # Query log audit & filter
│   │   └── upload_handler.go   # Upload file media (logo, TTD, bukti transfer)
│   ├── middleware/
│   │   ├── jwt_auth.go         # Verifikasi Bearer token JWT
│   │   ├── api_key.go          # Proteksi opsi integrasi eksternal
│   │   ├── cors.go             # Konfigurasi Cross-Origin Resource Sharing
│   │   └── rate_limiter.go     # Fiber rate limiter per IP
│   ├── repository/
│   │   └── repository.go       # GORM Data Access Layer & Seed Initial Data
│   └── service/
│       └── service.go          # Business Logic Engine & ACID Cascade Transactions
├── pkg/
│   ├── response/
│   │   └── response.go         # Standarisasi JSON: Success(c, ...) & Error(c, ...)
│   └── security/
│       └── security.go         # Hashing Bcrypt & JWT Generator/Parser
├── uploads/                    # Direktori penyimpanan statis aset gambar
├── .env.example
├── .gitignore
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

### 2.2 Model Entitas Database (GORM - `internal/domain/entity.go`)
```go
package domain

import (
	"time"
	"gorm.io/gorm"
)

// Profil Kreator & Pengaturan Studio (Owner 1 : N Entitas Lainnya)
type User struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Email              string         `gorm:"size:150;uniqueIndex;not null" json:"email"`
	PasswordHash       string         `gorm:"size:255;not null" json:"-"`
	Name               string         `gorm:"size:150;not null" json:"name"`
	AvatarURL          string         `gorm:"type:text" json:"avatar_url"`
	Company            string         `gorm:"size:200" json:"company"`
	Phone              string         `gorm:"size:50" json:"phone"`
	CompanyAddress     string         `gorm:"type:text" json:"company_address"`
	NPWP               string         `gorm:"size:50" json:"npwp"`
	TaxRate            float64        `gorm:"default:11.0" json:"tax_rate"` // Default PPN (0 - 100)
	LogoURL            string         `gorm:"type:text" json:"logo_url"`
	SignatureURL       string         `gorm:"type:text" json:"signature_url"`
	BankName           string         `gorm:"size:100" json:"bank_name"`
	BankAccountNumber  string         `gorm:"size:100" json:"bank_account_number"`
	BankAccountHolder  string         `gorm:"size:150" json:"bank_account_holder"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

// Entitas Tugas (Endorse, Barter, Lainnya)
type Task struct {
	ID                uint            `gorm:"primaryKey" json:"id"`
	UserID            uint            `gorm:"index;not null" json:"user_id"`
	Type              string          `gorm:"size:50;not null" json:"type"` // endorse | barter | other
	Title             string          `gorm:"size:255;not null" json:"title"`
	Brand             string          `gorm:"size:150;not null" json:"brand"`
	FeeOrValue        int64           `gorm:"default:0" json:"fee_or_value"` // Rupiah integer (anti-floating error)
	DueDate           time.Time       `gorm:"not null" json:"due_date"`
	Deliverable       string          `gorm:"type:text" json:"deliverable"`
	Platform          string          `gorm:"size:100" json:"platform"`
	ShipmentNumber    string          `gorm:"size:100" json:"shipment_number"`
	PicName           string          `gorm:"size:150" json:"pic_name"`
	PicPhone          string          `gorm:"size:50" json:"pic_phone"`
	Priority          string          `gorm:"size:20;default:'medium'" json:"priority"` // high | medium | low
	Status            string          `gorm:"size:50;default:'draft'" json:"status"`   // draft | waiting_payment | paid | posting | completed
	Brief             string          `gorm:"type:text" json:"brief"`
	PaymentProofURL   string          `gorm:"type:text" json:"payment_proof_url"`
	PaymentProofName  string          `gorm:"size:255" json:"payment_proof_name"`
	RelatedInvoiceID  *uint           `gorm:"index" json:"related_invoice_id"`
	Checklist         []ChecklistItem `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"checklist"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	DeletedAt         gorm.DeletedAt  `gorm:"index" json:"-"`
}

// Item Tahapan Checklist Tugas
type ChecklistItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    uint      `gorm:"index;not null" json:"task_id"`
	Text      string    `gorm:"size:255;not null" json:"text"`
	Completed bool      `gorm:"default:false" json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Entitas Tagihan Resmi (Invoice)
type Invoice struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"index;not null" json:"user_id"`
	RelatedTaskID     *uint          `gorm:"index" json:"related_task_id"`
	InvoiceNumber     string         `gorm:"size:100;uniqueIndex;not null" json:"invoice_number"`
	ClientName        string         `gorm:"size:200;not null" json:"client_name"`
	ClientEmail       string         `gorm:"size:150" json:"client_email"`
	ClientPhone       string         `gorm:"size:50" json:"client_phone"`
	ClientAddress     string         `gorm:"type:text" json:"client_address"`
	IssueDate         time.Time      `gorm:"not null" json:"issue_date"`
	DueDate           time.Time      `gorm:"not null" json:"due_date"`
	IncludeTax        bool           `gorm:"default:true" json:"include_tax"`
	TaxRate           float64        `gorm:"default:11.0" json:"tax_rate"` // Persentase pajak, misal 11.0
	Subtotal          int64          `gorm:"default:0" json:"subtotal"`
	TaxAmount         int64          `gorm:"default:0" json:"tax_amount"`
	GrandTotal        int64          `gorm:"default:0" json:"grand_total"`
	Status            string         `gorm:"size:30;default:'unpaid'" json:"status"` // unpaid | paid
	PaymentProofURL   string         `gorm:"type:text" json:"payment_proof_url"`
	PaymentProofName  string         `gorm:"size:255" json:"payment_proof_name"`
	Items             []InvoiceItem  `gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE" json:"items"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// Rincian Layanan per Invoice
type InvoiceItem struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	InvoiceID   uint      `gorm:"index;not null" json:"invoice_id"`
	Description string    `gorm:"size:255;not null" json:"description"`
	Quantity    int       `gorm:"default:1" json:"quantity"`
	UnitPrice   int64     `gorm:"default:0" json:"unit_price"`
	Total       int64     `gorm:"default:0" json:"total"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Buku Kas / Cashflow (Pemasukan & Biaya)
type Transaction struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"index;not null" json:"user_id"`
	RelatedInvoiceID  *uint          `gorm:"uniqueIndex" json:"related_invoice_id"` // UNIQUE: fisik cegah duplikasi pelunasan
	Type              string         `gorm:"size:20;not null" json:"type"`          // income | expense
	Amount            int64          `gorm:"not null" json:"amount"`
	Description       string         `gorm:"type:text;not null" json:"description"`
	Category          string         `gorm:"size:100;not null" json:"category"`
	Date              time.Time      `gorm:"not null" json:"date"`
	AttachmentURL     string         `gorm:"type:text" json:"attachment_url"`
	AttachmentName    string         `gorm:"size:255" json:"attachment_name"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// Log Jejak Audit Kronologis
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Actor     string    `gorm:"size:150;not null" json:"actor"`
	Module    string    `gorm:"size:50;not null" json:"module"` // Tugas | Invoice | Keuangan | Setting
	Action    string    `gorm:"size:100;not null" json:"action"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Detail    string    `gorm:"type:text" json:"detail"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}
```

### 2.3 Mekanisme Cascade Atomik (ACID Transaction)
Ketika aksi pelunasan dipicu (baik melalui `POST /api/v1/creator/tasks/:id/pay` atau `POST /api/v1/creator/invoices/:id/pay`), service layer mengeksekusi dalam satu transaksi database `db.Transaction`:
```go
func (s *serviceImpl) MarkTaskPaid(userID uint, taskID uint, req dto.PaymentConfirmationRequest) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Lock & ambil data tugas
		var task domain.Task
		if err := tx.Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
			return err
		}
		if task.Status == "paid" {
			return errors.New("tugas sudah lunas dan terkunci")
		}

		// 2. Cari atau Buat Invoice Terkait (BR-06)
		var invoice domain.Invoice
		var isNewInvoice bool
		if task.RelatedInvoiceID != nil {
			if err := tx.First(&invoice, *task.RelatedInvoiceID).Error; err != nil {
				return err
			}
		} else {
			isNewInvoice = true
			invNumber := s.generateInvoiceNumber(tx, "ENDORSE")
			userSetting, _ := s.getUserSettings(tx, userID)
			taxRate := userSetting.TaxRate
			subtotal := task.FeeOrValue
			taxAmount := math.Round(subtotal * (taxRate / 100.0))
			grandTotal := subtotal + taxAmount

			invoice = domain.Invoice{
				UserID:        userID,
				RelatedTaskID: &task.ID,
				InvoiceNumber: invNumber,
				ClientName:    task.Brand,
				IssueDate:     time.Now(),
				DueDate:       task.DueDate,
				IncludeTax:    true,
				TaxRate:       taxRate,
				Subtotal:      subtotal,
				TaxAmount:     taxAmount,
				GrandTotal:    grandTotal,
				Status:        "unpaid",
				Items: []domain.InvoiceItem{
					{Description: task.Title, Quantity: 1, UnitPrice: subtotal, Total: subtotal},
				},
			}
			if err := tx.Create(&invoice).Error; err != nil {
				return err
			}
			task.RelatedInvoiceID = &invoice.ID
		}

		// 3. Update Status Tugas & Invoice menjadi Lunas ('paid') + bukti bayar
		task.Status = "paid"
		task.PaymentProofURL = req.PaymentProofURL
		task.PaymentProofName = req.PaymentProofName
		if err := tx.Save(&task).Error; err != nil {
			return err
		}

		invoice.Status = "paid"
		invoice.PaymentProofURL = req.PaymentProofURL
		invoice.PaymentProofName = req.PaymentProofName
		if err := tx.Save(&invoice).Error; err != nil {
			return err
		}

		// 4. Catat Pemasukan Kas Otomatis Terkunci (BR-07, BR-08, BR-09)
		cashflow := domain.Transaction{
			UserID:           userID,
			RelatedInvoiceID: &invoice.ID,
			Type:             "income",
			Amount:           invoice.GrandTotal,
			Description:      fmt.Sprintf("Pelunasan Invoice %s (%s)", invoice.InvoiceNumber, task.Brand),
			Category:         "Pembayaran Endorse",
			Date:             time.Now(),
			AttachmentURL:    req.PaymentProofURL,
			AttachmentName:   req.PaymentProofName,
		}
		if err := tx.Create(&cashflow).Error; err != nil {
			return err
		}

		// 5. Catat Entri Audit Log
		audit := domain.AuditLog{
			UserID:    userID,
			Actor:     req.ActorName,
			Module:    "Tugas",
			Action:    "MARK_PAID",
			Title:     fmt.Sprintf("Pelunasan Tugas %s", task.Title),
			Detail:    fmt.Sprintf("Pelunasan invoice %s senilai Rp %.0f dan pencatatan kas otomatis", invoice.InvoiceNumber, invoice.GrandTotal),
			CreatedAt: time.Now(),
		}
		return tx.Create(&audit).Error
	})
}
```

### 2.4 Daftar Lengkap Endpoint REST API
| Modul | Method | Endpoint | Deskripsi & Aturan |
| :--- | :--- | :--- | :--- |
| **Auth** | `POST` | `/api/v1/auth/login` | Login menggunakan email & password. Mengembalikan JWT token. |
| **Auth** | `POST` | `/api/v1/auth/quick-login` | Akses instan profile demo (Reza Winata / Amanda) untuk pengujian. |
| **Upload** | `POST` | `/api/v1/upload` | Multipart form file (maks 2MB) -> mengembalikan `{ url, filename }`. |
| **Dashboard** | `GET` | `/api/v1/creator/dashboard` | Mengambil estimasi nilai proyek, kas bersih, 2 tugas agenda, tagihan pending. |
| **Tugas** | `GET` | `/api/v1/creator/tasks` | Query: `month`, `year`, `type` (semua/endorse/barter/other), `status`, `page`, `limit`. |
| **Tugas** | `POST` | `/api/v1/creator/tasks` | Membuat tugas baru dengan validasi form adaptif & catat `CREATE_TASK`. |
| **Tugas** | `GET` | `/api/v1/creator/tasks/:id` | Mengambil detail lengkap tugas, checklist, dan bukti bayar. |
| **Tugas** | `PUT` | `/api/v1/creator/tasks/:id/status` | Mengubah status non-pelunasan (draft, waiting_payment, posting, selesai). |
| **Tugas** | `POST` | `/api/v1/creator/tasks/:id/checklist/:itemId/toggle` | Toggle centang checklist pekerjaan secara real-time. |
| **Tugas** | `POST` | `/api/v1/creator/tasks/:id/pay` | Memicu cascade pelunasan: task -> invoice -> kas masuk -> audit log. |
| **Tugas** | `DELETE` | `/api/v1/creator/tasks/:id` | Hapus tugas. Ditolak jika tugas sudah berstatus `paid`. |
| **Invoice** | `GET` | `/api/v1/creator/invoices` | Query: `month`, `year`, `status` (semua/paid/unpaid), `page`, `limit`. |
| **Invoice** | `POST` | `/api/v1/creator/invoices` | Penerbitan invoice manual dengan item baris dinamis & sakelar PPN. |
| **Invoice** | `GET` | `/api/v1/creator/invoices/:id` | Mengambil detail rincian invoice dan status pembayaran. |
| **Invoice** | `PATCH` | `/api/v1/creator/invoices/:id/toggle-tax` | Toggle sakelar PPN cepat (hanya boleh jika status masih unpaid). |
| **Invoice** | `POST` | `/api/v1/creator/invoices/:id/pay` | Pelunasan tagihan invoice manual/otomatis + sinkronisasi cascade. |
| **Keuangan** | `GET` | `/api/v1/creator/cashflow` | Query: `month`, `year`, `type` (semua/income/expense), `page`, `limit`. |
| **Keuangan** | `POST` | `/api/v1/creator/cashflow` | Mencatat transaksi manual (pengeluaran atau kas masuk). |
| **Keuangan** | `DELETE` | `/api/v1/creator/cashflow/:id` | Menghapus transaksi kas manual. Ditolak keras jika `relatedInvoiceId != null`. |
| **Laporan** | `GET` | `/api/v1/creator/reports` | Mengambil rekapitulasi 4 KPI, daftar proyek gabungan, dan data cetak PDF. |
| **Pengaturan** | `GET` | `/api/v1/creator/settings` | Mengambil profil owner, identitas studio, logo, TTD digital, tarif PPN, & bank. |
| **Pengaturan** | `PUT` | `/api/v1/creator/settings` | Memperbarui profil & identitas studio (berlaku instan ke seluruh cetakan). |
| **Audit Log** | `GET` | `/api/v1/creator/audit-logs` | Query: `module`, `search`, `limit`, `page`. |

---

## 3. Arsitektur Frontend (`FE_PM` - Mengikuti Pola `FE_COMPRO`)

### 3.1 Struktur Direktori Frontend (Next.js 16 App Router)
```
FE_PM/
├── app/
│   ├── (auth)/
│   │   └── login/
│   │       └── page.tsx            # Layar Login + Switcher Akun Cepat
│   ├── (dashboard)/
│   │   ├── layout.tsx              # App Shell: Sticky Navbar + Floating Pill Dock
│   │   ├── beranda/
│   │   │   └── page.tsx            # Tab 1: Dashboard Utama
│   │   ├── tugas/
│   │   │   └── page.tsx            # Tab 2: Pusat Tugas (Endorse, Barter, Lainnya)
│   │   ├── laporan/
│   │   │   └── page.tsx            # Tab 3: Laporan, Rekapitulasi & Audit Log
│   │   ├── invoice/
│   │   │   └── page.tsx            # Tab 4: Generator Invoice & Pelunasan
│   │   ├── keuangan/
│   │   │   └── page.tsx            # Tab 5: Pencatatan Arus Kas & Biaya
│   │   └── pengaturan/
│   │       └── page.tsx            # Tab 6: Pengaturan Profil, PPN, TTD, & Bank
│   ├── globals.css                 # Token CSS, Utilities, @media print isolation
│   ├── layout.tsx                  # Root layout, fonts (Geist/Inter), toaster
│   └── page.tsx                    # Route redirect ke /beranda atau /login
├── components/
│   ├── shell/
│   │   ├── Navbar.tsx              # Header inisial CT, judul dinamis, avatar & keluar
│   │   └── BottomTabBar.tsx        # Floating Pill Dock frosted glass + real-time badge
│   ├── common/
│   │   ├── Pagination.tsx          # Navigasi paginasi 4-item (Prev, Number, Next)
│   │   ├── ConfirmDialog.tsx       # Dialog modal konfirmasi penghapusan & peringatan
│   │   ├── LightboxModal.tsx       # Fullscreen preview gambar bukti transfer / nota
│   │   └── AuditLogDrawer.tsx      # Bottom-sheet sliding drawer log aktivitas
│   ├── tugas/
│   │   ├── TaskCard.tsx            # Kartu tugas sesuai spesifikasi mockup
│   │   ├── TaskCreateModal.tsx     # Form adaptif berdasarkan tipe tugas
│   │   ├── TaskDetailModal.tsx     # 3 blok kontrol status, rincian, dan checklist
│   │   └── PaymentConfirmModal.tsx # Upload bukti transfer & konfirmasi cascade
│   ├── invoice/
│   │   ├── InvoiceCard.tsx         # Kartu invoice dengan toggle PPN instan
│   │   ├── InvoiceCreateModal.tsx  # Modal pembuatan invoice dengan item dinamis
│   │   └── InvoicePrintModal.tsx   # Modal pratinjau A4 resmi & download PDF
│   ├── keuangan/
│   │   ├── CashflowHero.tsx        # Kartu Arus Kas Bersih (hijau surplus / merah defisit)
│   │   ├── CashflowCard.tsx        # Kartu pemasukan / pengeluaran
│   │   └── TransactionCreateModal.tsx # Form catat biaya / pemasukan kas
│   ├── pengaturan/
│   │   ├── SignatureCanvas.tsx     # HTML5 Canvas sentuh (jari/stylus) + upload file
│   │   └── LogoUploader.tsx        # Upload logo studio dengan inisial fallback
│   └── pdf/
│       ├── ReportSheetOne.tsx      # Lembar 1 Laporan Rekapitulasi A4
│       ├── ReportSheetTwo.tsx      # Lembar 2 Lampiran Bukti Transfer A4
│       └── InvoiceSheetA4.tsx      # Dokumen Cetak Invoice Resmi A4
├── lib/
│   ├── api-client.ts               # HTTP Client wrapper (fetch + JWT Auth interceptor)
│   ├── formatters.ts               # Format Rupiah (Rp ...), format tanggal Indonesia
│   ├── pdf-generator.ts            # Utilitas render cetak biner & DOM isolation
│   └── types.ts                    # TypeScript interfaces & types lengkap
├── public/                         # Logo, favicon, ilustrasi default
├── package.json
├── postcss.config.mjs
├── tsconfig.json
└── tailwind.config.ts
```

### 3.2 Sistem Desain & Antarmuka Sesuai Mockup Web View
Sesuai tangkapan layar tampilan mockup awal:
1. **Palet Warna & Token Visual**:
   - Primary Accent: Indigo (`#4F46E5`, `bg-indigo-600`) untuk branding, CTA tombol, dan tab aktif.
   - Pemasukan & Status Lunas: Hijau Emerald (`#10B981`, `bg-emerald-500`, border `#34D399`).
   - Pengeluaran, Peringatan & Badge: Merah Rose (`#F43F5E`, `bg-rose-500`).
   - Kartu Hero Dashboard: Gradien gelap elegan (`linear-gradient(135deg, #1E1B4B 0%, #0F172A 100%)`).
   - Latar Belakang Aplikasi: Slate sejuk lembut (`#F8FAFC`).
2. **App Shell**:
   - **Sticky Top Header**: Tinggi 64px, logo `CT` berlatar indigo di kiri, nama modul dinamis di tengah (*Dashboard Utama*, *Pusat Tugas*, dll.), serta avatar owner dan tombol logout di kanan.
   - **Floating Pill Dock**: Kapsul bulat penuh (`rounded-full`) melayang di bagian bawah dengan efek kaca buram (`backdrop-blur-xl bg-white/80 border border-slate-200/80 shadow-2xl`).
   - **Badge Real-time Tab Invoice**: Menampilkan bulatan merah kecil bertuliskan jumlah tagihan yang belum lunas (`status !== 'paid'`). Badge otomatis hilang jika nol.
3. **Paginasi & Layout Batas 4 Item**:
   - Halaman Tugas, Laporan, Invoice, dan Keuangan mengimplementasikan pembatasan tegas maksimal 4 kartu per halaman (`BR-01`) dengan kontrol nomor halaman yang elegan.
4. **Respon Web View & Mobile Wrapper**:
   - Antarmuka dirancang fleksibel: tampil rapi dan fokus pada layar sempit/mobile (bottom-sheet ala iOS, pill dock sentuh) dan terpusat dengan kontainer bersih (`max-w-4xl mx-auto`) saat dibuka pada layar monitor desktop lebar (*Web View*).

---

## 4. Spesifikasi Mesin Cetak PDF & Isolasi Cetak (Bab 6 SRS)

### 4.1 Dokumen Cetak 1: Laporan Rekapitulasi (Maksimal 2 Lembar A4)
- **Lembar 1 (Ringkasan Kinerja & Rekapitulasi Proyek)**:
  - **Kop Surat Kiri**: Logo studio 48x48px (atau blok inisial studio jika logo kosong - BR-18), nama perusahaan bold 14pt, subjudul dan alamat lengkap.
  - **Kop Surat Kanan**: Periode laporan (misal: *Oktober 2026*), filter tipe kerja sama (*Endorse + Barter*), dan NPWP studio.
  - **4 Kotak KPI**: Total Proyek, Total Nilai Fee (termasuk PPN - BR-14), Sudah Dibayar, Belum Lunas.
  - **Tabel Rekap 7 Kolom**: `No` (5%), `Proyek / Brand` (31%), `Tipe` (10%), `No. Invoice` (18% mono), `Nilai IDR` (18% rata kanan), `Status` (10%), `Lampiran` (8% - nomor referensi bukti `#1`, `#2`, atau `-`).
  - **Baris Total & Pengesahan**: Total nilai keseluruhan, jumlah lunas vs pending, tanggal cetak, dan **Tanda Tangan Digital** owner.
- **Lembar 2 (Lampiran Bukti Transfer Gambar)**:
  - Otomatis dimulai pada halaman baru menggunakan CSS `page-break-before: always;`.
  - Hanya dicetak jika opsi *Sertakan Lampiran Bukti Transfer Gambar* dicentang dan minimal ada 1 proyek yang memiliki foto bukti bayar (`BR-16`).
  - Layout Grid 2 Kolom (`grid-cols-2 gap-4`), masing-masing kartu memuat: nomor urut lampiran, nama proyek, nomor invoice, nilai transaksi, nama file, stempel terverifikasi, serta **foto bukti transfer asli** (`object-contain max-h-52`).

### 4.2 Dokumen Cetak 2: Invoice Resmi A4 (1 Lembar)
- **Kop Dokumen**: Identitas studio lengkap di sisi kiri; Judul `INVOICE`, nomor tagihan monospace, dan badge status (`LUNAS` / `MENUNGGU PEMBAYARAN`) di sisi kanan.
- **Grid Klien & Tanggal**: 2 kolom memuat nama klien, email, telepon, alamat penagihan di kiri; tanggal terbit, jatuh tempo, dan status PPN di kanan.
- **Tabel 4 Kolom**: Deskripsi pekerjaan (52%), Kuantitas (8%), Harga Satuan (20%), Total (20%).
- **Kotak Perhitungan Pajak**: Subtotal, PPN (`taxRate%` atau `Rp 0`), dan **Grand Total Tagihan**.
- **Footer**: Instruksi rekening transfer bank di sisi kiri; blok pengesahan tanda tangan digital di sisi kanan.
- **Lampiran Terverifikasi (Opsional)**: Jika invoice telah berstatus `paid` dan memiliki berkas bukti, thumbnail foto bukti transfer tercetak rapi di bagian kaki dokumen.

### 4.3 Teknik Isolasi Cetak Browser (`@media print`)
```css
@media print {
  body * {
    visibility: hidden;
  }
  #printable-report-document,
  #printable-report-document *,
  #printable-invoice-modal,
  #printable-invoice-modal * {
    visibility: visible;
  }
  #printable-report-document,
  #printable-invoice-modal {
    position: absolute;
    left: 0;
    top: 0;
    width: 210mm;
    margin: 0;
    padding: 10mm 12mm;
    background: #ffffff !important;
  }
  .no-print, header, nav, #floating-dock {
    display: none !important;
  }
  .page-break {
    page-break-before: always;
    break-before: page;
  }
  .avoid-break {
    page-break-inside: avoid;
    break-inside: avoid;
  }
}
```

---

## 5. Matriks Skenario Uji Penerimaan (UAT & Edge Cases)

| ID UAT | Kasus Uji | Skenario Tindakan | Hasil yang Diharapkan |
| :--- | :--- | :--- | :--- |
| **UAT-01** | Akses Autentikasi | Klik kartu Quick Login profil "Reza Winata". | Token JWT tersimpan, masuk ke Beranda, nama & avatar tampil di navbar. |
| **UAT-02** | Pengaturan Bisnis | Ubah nama studio, tarif PPN (12%), unggah logo & buat TTD kanvas. | Toast sukses, log `UPDATE_SETTINGS`, kop dokumen & navbar terbarui instan. |
| **UAT-03** | Tambah Tugas Endorse | Buat tugas Endorse Rp 15.000.000 dengan 3 tahapan checklist. | Kartu berstatus `Draft`, log `CREATE_TASK`, muncul di agenda Beranda. |
| **UAT-04** | Pelunasan Tugas (Cascade) | Ubah status tugas ke `Sudah Dibayar`, upload bukti transfer & konfirmasi. | (1) Tugas jadi `paid` + bukti;<br>(2) Invoice Rp 16.650.000 (PPN 11%) terbit `paid`;<br>(3) Kas masuk Rp 16.650.000 tercatat terkunci;<br>(4) Badge notifikasi tidak bertambah. |
| **UAT-05** | Barter Tanpa Tagihan | Buat tugas Barter (valuasi Rp 1.500.000), ubah ke `Posting`. | Tidak ada invoice terbit, saldo kas tidak berubah, log tercatat. |
| **UAT-06** | Invoice Manual & PPN | Buat invoice 2 baris layanan, coba matikan lalu hidupkan toggle PPN. | Subtotal, PPN, dan Grand Total dihitung ulang instan; badge invoice +1. |
| **UAT-07** | Pelunasan dari Invoice | Klik tombol `Tandai Lunas` pada kartu invoice manual + upload bukti. | Invoice status `paid`, kas masuk bertambah, badge invoice berkurang 1. |
| **UAT-08** | Proteksi Hapus Kas | Coba cari tombol hapus pada kas hasil pelunasan invoice; hapus biaya manual. | Tombol hapus kas invoice tersembunyi/diblokir; biaya manual terhapus normal. |
| **UAT-09** | Filter & Paginasi Laporan | Pilih periode bulan & chip `Endorse + Barter`, lalu buka halaman 2. | 4 kartu pertama di hal 1 (#1 s/d #4), kartu ke-5 (#5) di hal 2; 4 KPI akurat. |
| **UAT-10** | Cetak Laporan PDF | Buka pratinjau laporan, centang lampiran gambar, klik Cetak PDF. | Output 2 lembar A4 rapi: Lembar 1 kop + tabel + TTD; Lembar 2 grid bukti. |
| **UAT-11** | Cetak Invoice PDF | Klik tombol pratinjau invoice -> Cetak PDF. | Dokumen A4 1 halaman lengkap dengan rincian, PPN, rekening, dan TTD. |
| **UAT-12** | Pencarian Audit Log | Buka drawer audit log, ketik keyword atau pilih chip modul `Invoice`. | Log tersaring instan sesuai filter kata kunci dan modul. |
| **UAT-13** | Paid Lock Protection | Coba turunkan status tugas `paid` kembali ke `draft` melalui API/UI. | Ditolak oleh sistem dengan error `Tugas sudah lunas dan terkunci`. |
| **UAT-14** | Delete Protection Task | Coba hapus tugas yang sudah lunas. | Tombol hapus terkunci/disabled; request API ditolak. |

---

## 6. Rencana Tahapan Eksekusi (Implementation Roadmap)

### Fase 1: Inisialisasi & Setup Arsitektur Backend (`BE_PM`)
- [x] Inisialisasi Go module (`be-pm`), instalasi dependencies (Fiber v2, GORM, SQLite/PostgreSQL driver, JWT v5, godotenv, crypto).
- [ ] Implementasi `config/config.go`, middleware (CORS, Rate Limiter, Logger, Recover, JWT Auth).
- [ ] Definisi entitas GORM (`internal/domain/entity.go`) & migrasi skema tabel otomatis.
- [ ] Pembuatan `repository/repository.go` dengan fungsi data seeder bawaan (`SeedInitialData`).
- [ ] Implementasi Service Layer (`internal/service/service.go`) mencakup mesin transaksi ACID cascade pelunasan dan generator nomor invoice otomatis.
- [ ] Implementasi seluruh Handlers HTTP & Response standardization (`pkg/response`).
- [ ] Pengujian API menggunakan unit/integration tests & validasi seluruh endpoint.

### Fase 2: Inisialisasi & Setup Arsitektur Frontend (`FE_PM`)
- [ ] Inisialisasi Next.js 16 (App Router) dengan TypeScript dan Tailwind CSS v4.
- [ ] Setup font Geist / Inter, sistem variabel warna, styling `@media print`, dan instalasi Lucide React.
- [ ] Implementasi `lib/api-client.ts` untuk komunikasi type-safe dengan Backend API.
- [ ] Pembuatan komponen inti App Shell: `Navbar.tsx` (sticky top) dan `BottomTabBar.tsx` (floating pill dock frosted glass dengan counter badge real-time).
- [ ] Implementasi komponen bersama: `Pagination.tsx`, `LightboxModal.tsx`, `ConfirmDialog.tsx`.

### Fase 3: Pembangunan Halaman 0, 1, dan 6 (Dasar & Konfigurasi)
- [ ] **Halaman 0 (Login)**: Pemilih profil cepat (Quick Login) & formulir login standar.
- [ ] **Halaman 6 (Pengaturan & Profil)**: Profil owner, identitas studio, logo uploader dengan inisial fallback, HTML5 Canvas tanda tangan digital sentuh/stylus, tarif PPN default, dan nomor rekening bank.
- [ ] **Halaman 1 (Beranda)**: Kartu Hero gradient gelap (estimasi nilai & kas), 3 pintasan aksi cepat, agenda 2 tugas, ringkasan cashflow, dan tagihan invoice pending.

### Fase 4: Pembangunan Modul Operasional (Tugas, Invoice, Keuangan)
- [ ] **Halaman 2 (Pusat Tugas)**: Filter bulan dinamis, tab 4 tipe tugas, status chips dinamis, kartu tugas terpaginasi 4-item, modal form adaptif, modal detail tugas dengan kontrol status terpusat, checklist interaktif, serta modal konfirmasi bayar & upload bukti transfer.
- [ ] **Halaman 4 (Generator Invoice)**: Filter status chips (Semua, Paid, Unpaid), kartu invoice dengan toggle PPN instan, modal buat invoice dengan item dinamis, modal konfirmasi lunas, dan modal cetak invoice A4.
- [ ] **Halaman 5 (Keuangan)**: Kartu Hero Arus Kas Bersih (hijau surplus / merah defisit), kartu total pemasukan & pengeluaran, riwayat transaksi terpaginasi dengan kunci hapus pada entri hasil invoice, dan modal catat transaksi biaya/kas.

### Fase 5: Modul Laporan, Audit Log, dan Engine Cetak PDF
- [ ] **Halaman 3 (Laporan & Rekapitulasi)**: Banner periode, filter bulan/tahun/tipe kerja sama, 4 kartu KPI, daftar kartu proyek terpaginasi 4-item dengan nomor indeks berurutan (#1, #2, dst.), dan Lightbox bukti bayar.
- [ ] **Audit Log Bottom Sheet**: Drawer meluncur dari bawah dengan drag handle, live search, filter modul, dan daftar kronologis lengkap.
- [ ] **Mesin Cetak PDF**: Lembar 1 Laporan Rekapitulasi A4, Lembar 2 Lampiran Bukti Transfer (grid 2 kolom), dan Dokumen Invoice Resmi A4 dengan isolasi CSS `@media print`.

### Fase 6: Validasi End-to-End, UAT, dan Dokumentasi Final
- [ ] Uji coba seluruh alur end-to-end (UAT-01 sampai UAT-14).
- [ ] Pengujian respon tampilan pada berbagai ukuran layar (mobile viewport dan desktop web view).
- [ ] Verifikasi hasil ekspor / cetak PDF di Chrome, Edge, dan Safari.
- [ ] Penyusunan panduan deployment (Docker & Docker Compose).
