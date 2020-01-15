<?php
/**
 * Konfigurasi dasar WordPress.
 *
 * Berkas ini berisi konfigurasi-konfigurasi berikut: Pengaturan MySQL, Awalan Tabel,
 * Kunci Rahasia, Bahasa WordPress, dan ABSPATH. Anda dapat menemukan informasi lebih
 * lanjut dengan mengunjungi Halaman Codex {@link http://codex.wordpress.org/Editing_wp-config.php
 * Menyunting wp-config.php}. Anda dapat memperoleh pengaturan MySQL dari web host Anda.
 *
 * Berkas ini digunakan oleh skrip penciptaan wp-config.php selama proses instalasi.
 * Anda tidak perlu menggunakan situs web, Anda dapat langsung menyalin berkas ini ke
 * "wp-config.php" dan mengisi nilai-nilainya.
 *
 * @package WordPress
 */

// ** Pengaturan MySQL - Anda dapat memperoleh informasi ini dari web host Anda ** //
/** Nama basis data untuk WordPress */
define( 'DB_NAME', 'wordpress' );

/** Nama pengguna basis data MySQL */
define( 'DB_USER', 'root' );

/** Kata sandi basis data MySQL */
define( 'DB_PASSWORD', 'password' );

/** Nama host MySQL */
define( 'DB_HOST', 'docker.for.mac.host.internal' );

/** Set Karakter Basis Data yang digunakan untuk menciptakan tabel basis data. */
define( 'DB_CHARSET', 'utf8mb4' );

/** Jenis Collate Basis Data. Jangan ubah ini jika ragu. */
define('DB_COLLATE', '');

/**#@+
 * Kunci Otentifikasi Unik dan Garam.
 *
 * Ubah baris berikut menjadi frase unik!
 * Anda dapat menciptakan frase-frase ini menggunakan {@link https://api.wordpress.org/secret-key/1.1/salt/ Layanan kunci-rahasia WordPress.org}
 * Anda dapat mengubah baris-baris berikut kapanpun untuk mencabut validasi seluruh cookies. Hal ini akan memaksa seluruh pengguna untuk masuk log ulang.
 *
 * @since 2.6.0
 */
define( 'AUTH_KEY',         'A{gOC]v&ns9U1qOzb1cPkxauU<2k,7? jo@gOX=:lSkN&/Xul4d6rXSYAT*kfwTB' );
define( 'SECURE_AUTH_KEY',  '=7g}od@ZZ?uxk0>,f5N0j{)b4t_VB,?J>I62:/pYxi_+f6q>j3k7 L24_DSTq?:a' );
define( 'LOGGED_IN_KEY',    '[F; #(dnsq465Zb)Uf~L$QQ.!AJ^fC^XX: +q8Xi+#,3@0gA4A|e?i&E_9g8Xn&~' );
define( 'NONCE_KEY',        'M;)p7.kNM4#.>}y3A)PnMjh_z6FO0{~k0N|ly^`~9cN~y=xW)aroHz1+44fS|%$h' );
define( 'AUTH_SALT',        'h0q!O-or_y 3KoQFY;6TNM{b[ZUCqaA}~JPq%?X|aPHJrB&TpJW_n$|H>v-l;TaD' );
define( 'SECURE_AUTH_SALT', '$!lW{6*0S|_/yNDX)tN~B2_^kpjkK!B-SJ<p3VmnXr04|B-2I^e8>NfE>>E_-Jm+' );
define( 'LOGGED_IN_SALT',   't:6cT{(>k(1nIDVBe(1-NtNY,EtlQm@yBsy%&&qOPUv=eDC=GyS/PXmBqX`=S!uH' );
define( 'NONCE_SALT',       ')hd.[tQL_52Kw6o%ne/JL/#nh OVPWpn[mM=5b|yuQteuX_6IP7#+aiZa#{N wPn' );

/**#@-*/

/**
 * Awalan Tabel Basis Data WordPress.
 *
 * Anda dapat memiliki beberapa instalasi di dalam satu basis data jika Anda memberikan awalan unik
 * kepada masing-masing tabel. Harap hanya masukkan angka, huruf, dan garis bawah!
 */
$table_prefix = 'wp_';

/**
 * Untuk pengembang: Moda pengawakutuan WordPress.
 *
 * Ubah ini menjadi "true" untuk mengaktifkan tampilan peringatan selama pengembangan.
 * Sangat disarankan agar pengembang plugin dan tema menggunakan WP_DEBUG
 * di lingkungan pengembangan mereka.
 */
define('WP_DEBUG', false);

/* Cukup, berhenti menyunting! Selamat ngeblog. */

/** Lokasi absolut direktori WordPress. */
if ( !defined('ABSPATH') )
	define('ABSPATH', dirname(__FILE__) . '/');

/** Menentukan variabel-variabel WordPress berkas-berkas yang disertakan. */
require_once(ABSPATH . 'wp-settings.php');
