/*
Navicat MySQL Data Transfer

Source Server         : 101.200.35.63（测试）
Source Server Version : 50643
Source Host           : 101.200.35.63:3306
Source Database       : comjia_merge

Target Server Type    : MYSQL
Target Server Version : 50643
File Encoding         : 65001

Date: 2019-06-28 11:15:29
*/

SET FOREIGN_KEY_CHECKS=0;

-- ----------------------------
-- Table structure for `sqlexplain`
-- ----------------------------
DROP TABLE IF EXISTS `sqlexplain`;
CREATE TABLE `sqlexplain` (
  `id` bigint(30) unsigned NOT NULL AUTO_INCREMENT,
  `expl` varchar(500) DEFAULT NULL,
  `route` varchar(200) DEFAULT NULL,
  `sql_info` varchar(5000) DEFAULT NULL,
  `explain` varchar(1000) DEFAULT NULL,
  `last_query_cost` float(20,0) DEFAULT NULL,
  `level` varchar(10) DEFAULT NULL,
  `score` int(10) DEFAULT NULL,
  `content` varchar(150) DEFAULT NULL,
  `branch` varchar(100) DEFAULT NULL,
  `db` varchar(100) DEFAULT NULL,
  `flag` tinyint(1) DEFAULT '0' COMMENT '0 老数据，1 新数据',
  `daytime` date DEFAULT NULL,
  `create_datetime` datetime DEFAULT NULL,
  `status` tinyint(1) DEFAULT '0' COMMENT '0 显示 1 删除',
  PRIMARY KEY (`id`),
  KEY `idx_Level` (`level`) USING BTREE,
  KEY `idx_score` (`score`) USING BTREE,
  KEY `idx_branch` (`branch`) USING BTREE,
  KEY `idx_route` (`route`(191)) USING BTREE,
  KEY `idx_db` (`db`) USING BTREE,
  KEY `idx_create_time` (`create_datetime`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

