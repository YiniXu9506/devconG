 CREATE TABLE `phrase_click_models` (
  `id` bigint(20) NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */,
  `group_id` bigint(20) DEFAULT NULL,
  `open_id` longtext DEFAULT NULL,
  `phrase_id` bigint(20) DEFAULT NULL,
  `clicks` bigint(20) DEFAULT NULL,
  `click_time` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
  KEY `idx_phrase_clicks` (`phrase_id`,`group_id`,`clicks`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin /*T![auto_rand_base] AUTO_RANDOM_BASE=6510002 */


CREATE TABLE `phrase_models` (
  `phrase_id` bigint(20) NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */,
  `text` varchar(60) DEFAULT NULL,
  `group_id` bigint(20) DEFAULT NULL,
  `open_id` longtext DEFAULT NULL,
  `status` bigint(20) DEFAULT NULL,
  `create_time` bigint(20) DEFAULT NULL,
  `update_time` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`phrase_id`) /*T![clustered_index] CLUSTERED */,
  UNIQUE KEY `text` (`text`),
  KEY `idx_phrase_models_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin /*T![auto_rand_base] AUTO_RANDOM_BASE=4855472