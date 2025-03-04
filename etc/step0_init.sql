-- MySQL dump 10.13  Distrib 8.0.40, for Linux (x86_64)
--
-- Host: localhost    Database: gotest
-- ------------------------------------------------------
-- Server version	8.0.40-0ubuntu0.20.04.1

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Temporary view structure for view `ViewGroupSlot`
--

DROP TABLE IF EXISTS `ViewGroupSlot`;
/*!50001 DROP VIEW IF EXISTS `ViewGroupSlot`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `ViewGroupSlot` AS SELECT 
 1 AS `pub_id`,
 1 AS `site_id`,
 1 AS `slot_id`,
 1 AS `adv_id`,
 1 AS `campaign_id`,
 1 AS `item_id`,
 1 AS `cost_type`,
 1 AS `cost`,
 1 AS `endx`*/;
SET character_set_client = @saved_cs_client;

--
-- Temporary view structure for view `ViewSlot`
--

DROP TABLE IF EXISTS `ViewSlot`;
/*!50001 DROP VIEW IF EXISTS `ViewSlot`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `ViewSlot` AS SELECT 
 1 AS `pub_id`,
 1 AS `pub_name`,
 1 AS `site_id`,
 1 AS `site_name`,
 1 AS `site_url`,
 1 AS `slot_id`,
 1 AS `slot_name`,
 1 AS `item_id`,
 1 AS `item_name`,
 1 AS `cost_type`,
 1 AS `cost`,
 1 AS `startx`,
 1 AS `endx`,
 1 AS `campaign_id`,
 1 AS `campaign_name`,
 1 AS `adv_id`,
 1 AS `adv_name`,
 1 AS `ac_id`,
 1 AS `entitytype_id`,
 1 AS `entity_id`,
 1 AS `othertype_id`,
 1 AS `other_id`*/;
SET character_set_client = @saved_cs_client;

--
-- Temporary view structure for view `ViewSlotOpen`
--

DROP TABLE IF EXISTS `ViewSlotOpen`;
/*!50001 DROP VIEW IF EXISTS `ViewSlotOpen`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `ViewSlotOpen` AS SELECT 
 1 AS `pub_id`,
 1 AS `pub_name`,
 1 AS `site_id`,
 1 AS `site_name`,
 1 AS `site_url`,
 1 AS `slot_id`,
 1 AS `slot_name`,
 1 AS `item_id`,
 1 AS `item_name`,
 1 AS `cost_type`,
 1 AS `cost`,
 1 AS `startx`,
 1 AS `endx`,
 1 AS `campaign_id`,
 1 AS `campaign_name`,
 1 AS `adv_id`,
 1 AS `adv_name`,
 1 AS `ac_id`,
 1 AS `entitytype_id`,
 1 AS `entity_id`,
 1 AS `othertype_id`,
 1 AS `other_id`*/;
SET character_set_client = @saved_cs_client;

--
-- Temporary view structure for view `ViewTaoSlot`
--

DROP TABLE IF EXISTS `ViewTaoSlot`;
/*!50001 DROP VIEW IF EXISTS `ViewTaoSlot`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `ViewTaoSlot` AS SELECT 
 1 AS `slot_id`,
 1 AS `adv_id`,
 1 AS `campaign_id`,
 1 AS `item_id`,
 1 AS `cost_type`,
 1 AS `price`,
 1 AS `endx`,
 1 AS `cpm_fc`,
 1 AS `cpm_length`,
 1 AS `cpm_throttle`,
 1 AS `cpc_fc`,
 1 AS `cpc_length`*/;
SET character_set_client = @saved_cs_client;

--
-- Table structure for table `ac`
--

DROP TABLE IF EXISTS `ac`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ac` (
  `ac_id` int unsigned NOT NULL AUTO_INCREMENT,
  `entitytype_id` tinyint unsigned NOT NULL,
  `entity_id` int unsigned NOT NULL,
  `othertype_id` tinyint unsigned NOT NULL,
  `other_id` int unsigned NOT NULL,
  PRIMARY KEY (`ac_id`),
  UNIQUE KEY `type_id` (`entitytype_id`,`entity_id`,`othertype_id`,`other_id`),
  KEY `other` (`othertype_id`,`other_id`),
  CONSTRAINT `ac_ibfk_1` FOREIGN KEY (`entitytype_id`) REFERENCES `def_entitytype` (`entitytype_id`) ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT `ac_ibfk_2` FOREIGN KEY (`othertype_id`) REFERENCES `def_entitytype` (`entitytype_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ac`
--

LOCK TABLES `ac` WRITE;
/*!40000 ALTER TABLE `ac` DISABLE KEYS */;
/*!40000 ALTER TABLE `ac` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_ac_insert` AFTER INSERT ON `ac` FOR EACH ROW BEGIN
	INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
	VALUES (NEW.entitytype_id, NEW.entity_id, "ac", NOW());
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_ac_delete` AFTER DELETE ON `ac` FOR EACH ROW BEGIN
	INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
	VALUES (OLD.entitytype_id, OLD.entity_id, "ac_", NOW());
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `add_address`
--

DROP TABLE IF EXISTS `add_address`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `add_address` (
  `address_id` int unsigned NOT NULL AUTO_INCREMENT,
  `company` varchar(255) DEFAULT NULL,
  `contact` varchar(255) DEFAULT NULL,
  `contact_email` varchar(255) DEFAULT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `fax` varchar(255) DEFAULT NULL,
  `url` varchar(255) DEFAULT NULL,
  `street` varchar(255) DEFAULT NULL,
  `city` varchar(255) DEFAULT NULL,
  `state_id` int unsigned DEFAULT NULL,
  `zip` varchar(32) DEFAULT NULL,
  `country_id` int unsigned DEFAULT NULL,
  `ip` varchar(15) DEFAULT NULL,
  `created` datetime DEFAULT NULL,
  PRIMARY KEY (`address_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `add_address`
--

LOCK TABLES `add_address` WRITE;
/*!40000 ALTER TABLE `add_address` DISABLE KEYS */;
/*!40000 ALTER TABLE `add_address` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin`
--

DROP TABLE IF EXISTS `admin`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin` (
  `admin_id` int unsigned NOT NULL AUTO_INCREMENT,
  `login` varchar(255) NOT NULL,
  `passwd` varchar(255) NOT NULL,
  `active` enum('Yes','No','Pause') DEFAULT 'No',
  `created` datetime NOT NULL,
  PRIMARY KEY (`admin_id`),
  KEY `login` (`login`(8))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin`
--

LOCK TABLES `admin` WRITE;
/*!40000 ALTER TABLE `admin` DISABLE KEYS */;
INSERT INTO `admin` VALUES (101,'peter','534fd6f2e77a495d17bd2238bc1b0d66e5d5e4e7','Yes','2020-07-19 16:39:43');
/*!40000 ALTER TABLE `admin` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `adv`
--

DROP TABLE IF EXISTS `adv`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv` (
  `adv_id` int unsigned NOT NULL,
  `email` varchar(255) NOT NULL,
  `passwd` char(40) NOT NULL,
  `firstname` varchar(255) DEFAULT NULL,
  `lastname` varchar(255) DEFAULT NULL,
  `balance` float DEFAULT '0',
  `allow_debt` enum('Yes','No') DEFAULT 'No',
  `timezone_id` tinyint unsigned DEFAULT NULL,
  `address_id` int unsigned DEFAULT NULL,
  `active` enum('Yes','No','New') DEFAULT 'New',
  `access_order` enum('White','Black') DEFAULT 'Black',
  `created` datetime DEFAULT NULL,
  `updated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`adv_id`),
  UNIQUE KEY `email` (`email`(20)),
  KEY `address_id` (`address_id`),
  CONSTRAINT `adv_ibfk_1` FOREIGN KEY (`address_id`) REFERENCES `add_address` (`address_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv`
--

LOCK TABLES `adv` WRITE;
/*!40000 ALTER TABLE `adv` DISABLE KEYS */;
/*!40000 ALTER TABLE `adv` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_adv` AFTER UPDATE ON `adv` FOR EACH ROW BEGIN
  IF ((NEW.active <=> OLD.active) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
      VALUES (4, NEW.adv_id, NEW.active, NOW());
  END IF;
  IF ((NEW.access_order <=> OLD.access_order) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
      VALUES (4, NEW.adv_id, 'bw', NOW());
  END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `adv_attrname`
--

DROP TABLE IF EXISTS `adv_attrname`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv_attrname` (
  `attrname_id` int unsigned NOT NULL,
  `adv_id` int unsigned DEFAULT NULL,
  `attrname` varchar(255) NOT NULL,
  PRIMARY KEY (`attrname_id`),
  UNIQUE KEY `adv_id` (`adv_id`,`attrname`),
  CONSTRAINT `adv_attrname_ibfk_1` FOREIGN KEY (`adv_id`) REFERENCES `adv` (`adv_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv_attrname`
--

LOCK TABLES `adv_attrname` WRITE;
/*!40000 ALTER TABLE `adv_attrname` DISABLE KEYS */;
INSERT INTO `adv_attrname` VALUES (1401,NULL,'001001'),(1402,NULL,'001002'),(1403,NULL,'001003'),(1404,NULL,'001004'),(1405,NULL,'001005'),(1406,NULL,'001006'),(1407,NULL,'001007'),(1408,NULL,'001008'),(1409,NULL,'001009'),(1410,NULL,'001010'),(1411,NULL,'001011'),(1412,NULL,'001012'),(1413,NULL,'001013'),(1414,NULL,'001014'),(1415,NULL,'001015'),(1416,NULL,'001016'),(1111,NULL,'areacode'),(1108,NULL,'bandwidth'),(3,NULL,'Bmonth'),(94,NULL,'Book'),(6,NULL,'Bplace'),(74,NULL,'Brand'),(98,NULL,'Browser'),(1201,NULL,'browser'),(1202,NULL,'bversion'),(2,NULL,'Byear'),(89,NULL,'Car'),(1305,NULL,'child'),(1104,NULL,'city'),(88,NULL,'Commute'),(1101,NULL,'continent'),(1102,NULL,'country'),(1300,NULL,'demography'),(1206,NULL,'device'),(1105,NULL,'dma'),(1308,NULL,'education'),(1307,NULL,'ethnicity'),(86,NULL,'Finance'),(90,NULL,'Food'),(1001,NULL,'fullday'),(1002,NULL,'fullhour'),(82,NULL,'Game'),(1301,NULL,'gender'),(92,NULL,'Grocery'),(79,NULL,'Group'),(84,NULL,'Health'),(80,NULL,'Hold'),(4,NULL,'Horoscope'),(1306,NULL,'household'),(1304,NULL,'income'),(1107,NULL,'isp'),(85,NULL,'Learn'),(40,NULL,'Living'),(1303,NULL,'married'),(97,NULL,'Media'),(1309,NULL,'occupation'),(1203,NULL,'os'),(99,NULL,'Other'),(1204,NULL,'oversion'),(91,NULL,'Photo'),(78,NULL,'Plan'),(1205,NULL,'platform'),(77,NULL,'Price'),(1200,NULL,'pzua'),(95,NULL,'Report'),(75,NULL,'Screen'),(1,NULL,'Sex'),(83,NULL,'Shop'),(96,NULL,'Social'),(1103,NULL,'state'),(76,NULL,'Time'),(87,NULL,'Travel'),(81,NULL,'Vip'),(1003,NULL,'weekday'),(1004,NULL,'weekhour'),(93,NULL,'Work'),(1302,NULL,'yob'),(1106,NULL,'zip'),(5,NULL,'Zodiac');
/*!40000 ALTER TABLE `adv_attrname` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `adv_attrvalue`
--

DROP TABLE IF EXISTS `adv_attrvalue`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv_attrvalue` (
  `attrvalue_id` int unsigned NOT NULL AUTO_INCREMENT,
  `attrname_id` int unsigned NOT NULL,
  `value` varchar(255) NOT NULL,
  PRIMARY KEY (`attrvalue_id`),
  KEY `attrname_id` (`attrname_id`),
  CONSTRAINT `adv_attrvalue_ibfk_1` FOREIGN KEY (`attrname_id`) REFERENCES `adv_attrname` (`attrname_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv_attrvalue`
--

LOCK TABLES `adv_attrvalue` WRITE;
/*!40000 ALTER TABLE `adv_attrvalue` DISABLE KEYS */;
/*!40000 ALTER TABLE `adv_attrvalue` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `adv_balance`
--

DROP TABLE IF EXISTS `adv_balance`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv_balance` (
  `balance_id` int unsigned NOT NULL AUTO_INCREMENT,
  `limit_spend` float DEFAULT NULL,
  `limit_imp` int unsigned DEFAULT NULL,
  `limit_cli` int unsigned DEFAULT NULL,
  `current_spend` float DEFAULT '0',
  `current_imp` int unsigned DEFAULT '0',
  `current_cli` int unsigned DEFAULT '0',
  `created` datetime DEFAULT NULL,
  PRIMARY KEY (`balance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv_balance`
--

LOCK TABLES `adv_balance` WRITE;
/*!40000 ALTER TABLE `adv_balance` DISABLE KEYS */;
/*!40000 ALTER TABLE `adv_balance` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_balance` AFTER UPDATE ON `adv_balance` FOR EACH ROW BEGIN
  IF ((NEW.limit_spend <=> OLD.limit_spend) = 0) THEN
    INSERT INTO his_balance (balance_id, budget_old, budget_new, budget_add, created)
    VALUES (NEW.balance_id, OLD.limit_spend, NEW.limit_spend, NEW.limit_spend-OLD.limit_spend, NOW());
  END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `adv_campaign`
--

DROP TABLE IF EXISTS `adv_campaign`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv_campaign` (
  `campaign_id` int unsigned NOT NULL AUTO_INCREMENT,
  `adv_id` int unsigned NOT NULL,
  `campaign_name` varchar(255) NOT NULL,
  `foreign_id` varchar(255) DEFAULT NULL,
  `total_balance_id` int unsigned DEFAULT NULL,
  `daily_balance_id` int unsigned DEFAULT NULL,
  `access_order` enum('White','Black','Inherit') DEFAULT 'Inherit',
  `active` enum('New','Pass2','Yes','No','Pause') DEFAULT 'New',
  `created` datetime DEFAULT NULL,
  PRIMARY KEY (`campaign_id`),
  KEY `adv_id` (`adv_id`),
  KEY `total_balance_id` (`total_balance_id`),
  KEY `daily_balance_id` (`daily_balance_id`),
  CONSTRAINT `adv_campaign_ibfk_1` FOREIGN KEY (`adv_id`) REFERENCES `adv` (`adv_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `adv_campaign_ibfk_2` FOREIGN KEY (`total_balance_id`) REFERENCES `adv_balance` (`balance_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `adv_campaign_ibfk_3` FOREIGN KEY (`daily_balance_id`) REFERENCES `adv_balance` (`balance_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv_campaign`
--

LOCK TABLES `adv_campaign` WRITE;
/*!40000 ALTER TABLE `adv_campaign` DISABLE KEYS */;
/*!40000 ALTER TABLE `adv_campaign` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_campaign` AFTER UPDATE ON `adv_campaign` FOR EACH ROW BEGIN
  IF ((NEW.active <=> OLD.active) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
      VALUES (41, NEW.campaign_id, NEW.active, NOW());
  END IF;
  IF ((NEW.access_order <=> OLD.access_order) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
      VALUES (41, NEW.campaign_id, 'bw', NOW());
  END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `adv_creative`
--

DROP TABLE IF EXISTS `adv_creative`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv_creative` (
  `creative_id` int unsigned NOT NULL AUTO_INCREMENT,
  `creative_name` varchar(255) NOT NULL,
  `item_id` int unsigned DEFAULT NULL,
  `content` text,
  `weight` float DEFAULT '0.5',
  `active` enum('Yes','No') DEFAULT 'Yes',
  `created` datetime DEFAULT NULL,
  `updated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`creative_id`),
  KEY `item_id` (`item_id`),
  CONSTRAINT `adv_creative_ibfk_1` FOREIGN KEY (`item_id`) REFERENCES `adv_item` (`item_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv_creative`
--

LOCK TABLES `adv_creative` WRITE;
/*!40000 ALTER TABLE `adv_creative` DISABLE KEYS */;
/*!40000 ALTER TABLE `adv_creative` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_creative_insert` AFTER INSERT ON `adv_creative` FOR EACH ROW BEGIN
        INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
        VALUES (43, NEW.creative_id, "creative", NOW());
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_creative` AFTER UPDATE ON `adv_creative` FOR EACH ROW BEGIN
  IF ((NEW.active <=> OLD.active) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
      VALUES (43, NEW.creative_id, NEW.active, NOW());
  END IF;
  IF (((NEW.content <=> OLD.content) = 0) || ((NEW.weight <=> OLD.weight) = 0)) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
      VALUES (43, NEW.creative_id, "content", NOW());
  END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_creative_delete` AFTER DELETE ON `adv_creative` FOR EACH ROW BEGIN
        INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
        VALUES (43, OLD.creative_id, "creative_", NOW());
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `adv_ip`
--

DROP TABLE IF EXISTS `adv_ip`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv_ip` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `ip` int unsigned NOT NULL,
  `email` varchar(255) NOT NULL,
  `updated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `ret` enum('fail','success') NOT NULL DEFAULT 'fail',
  PRIMARY KEY (`id`),
  KEY `updated` (`updated`),
  KEY `ip` (`ip`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv_ip`
--

LOCK TABLES `adv_ip` WRITE;
/*!40000 ALTER TABLE `adv_ip` DISABLE KEYS */;
/*!40000 ALTER TABLE `adv_ip` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `adv_item`
--

DROP TABLE IF EXISTS `adv_item`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv_item` (
  `item_id` int unsigned NOT NULL AUTO_INCREMENT,
  `campaign_id` int unsigned DEFAULT NULL,
  `item_name` varchar(255) NOT NULL,
  `item_click` varchar(255) DEFAULT NULL,
  `size_id` int unsigned NOT NULL,
  `cost_type` enum('ROI','CPM','CPC','CPA') DEFAULT 'CPC',
  `cost` double DEFAULT NULL,
  `total_balance_id` int unsigned DEFAULT NULL,
  `daily_balance_id` int unsigned DEFAULT NULL,
  `startx` datetime DEFAULT NULL,
  `endx` datetime DEFAULT NULL,

  `cpm_fc` smallint unsigned DEFAULT NULL,
  `cpm_length` int unsigned DEFAULT NULL,
  `cpm_throttle` int unsigned DEFAULT NULL,
  `cpc_fc` smallint unsigned DEFAULT NULL,
  `cpc_length` int unsigned DEFAULT NULL,
  `page_cap` tinyint unsigned DEFAULT '2',
  
  `qa_item` int unsigned DEFAULT '0',
  `fl_slot` int unsigned DEFAULT '0',
  `fl_language` set("EN","ES","RU","DE","FR","JA","PT","TR","IT","FA","NL","PL","ZH","VI","ID","CS","KO","UK","AR","EL","FI","HE","SV","RO","HU","TH","DA","SK","BG","SR","NB","Other") DEFAULT 'EN',
  `fl_device` set('0','1','2','3','4','5','6','7') DEFAULT '0,1,2,3,4,5,6,7',
  `fl_position` set('0','1','2','3','4','5','6','7') DEFAULT '0,1,2,3,4,5,6,7',
  `qa_creative` enum('0','1','2','3','4','5','6','7','8','9','10','11','12','13','14','15','16','17','18') DEFAULT '0',
  `qa_expnd` enum('0','1','2','3','4','5') DEFAULT '0',
  `qa_mime` enum('0','1','2','3','4') DEFAULT '4',

  `channel_order` enum('White','Black') DEFAULT 'Black',
  `active` enum('Prepare','New','Pass2','Yes','No','Pause') DEFAULT 'Prepare',
  `created` datetime DEFAULT NULL,
  PRIMARY KEY (`item_id`),
  KEY `size_id` (`size_id`),
  KEY `campaign_id` (`campaign_id`),
  KEY `total_balance_id` (`total_balance_id`),
  KEY `daily_balance_id` (`daily_balance_id`),
  CONSTRAINT `adv_item_ibfk_1` FOREIGN KEY (`campaign_id`) REFERENCES `adv_campaign` (`campaign_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `adv_item_ibfk_2` FOREIGN KEY (`total_balance_id`) REFERENCES `adv_balance` (`balance_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `adv_item_ibfk_3` FOREIGN KEY (`daily_balance_id`) REFERENCES `adv_balance` (`balance_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv_item`
--

LOCK TABLES `adv_item` WRITE;
/*!40000 ALTER TABLE `adv_item` DISABLE KEYS */;
/*!40000 ALTER TABLE `adv_item` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_item` AFTER UPDATE ON `adv_item` FOR EACH ROW BEGIN
  IF ((NEW.active <=> OLD.active) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
      VALUES (42, NEW.item_id, NEW.active, NOW());
  END IF;
  IF ((NEW.channel_order <=> OLD.channel_order) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
      VALUES (41, NEW.campaign_id, 'channel', NOW());
  END IF;
  IF ((NEW.item_click <=> OLD.item_click) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
      VALUES (42, NEW.item_id, "landing", NOW());
  END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `adv_media`
--

DROP TABLE IF EXISTS `adv_media`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv_media` (
  `media_id` int unsigned NOT NULL AUTO_INCREMENT,
  `creative_id` int unsigned NOT NULL,
  `series` tinyint unsigned DEFAULT '0',
  `media` varchar(255) NOT NULL,
  `disk` varchar(255) NOT NULL,
  `mime` varchar(255) DEFAULT NULL,
  `size_id` int unsigned DEFAULT NULL,
  `created` datetime DEFAULT NULL,
  PRIMARY KEY (`media_id`),
  KEY `creative_id` (`creative_id`),
  CONSTRAINT `adv_media_ibfk_1` FOREIGN KEY (`creative_id`) REFERENCES `adv_creative` (`creative_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv_media`
--

LOCK TABLES `adv_media` WRITE;
/*!40000 ALTER TABLE `adv_media` DISABLE KEYS */;
/*!40000 ALTER TABLE `adv_media` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `adv_targetname`
--

DROP TABLE IF EXISTS `adv_targetname`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv_targetname` (
  `targetname_id` int unsigned NOT NULL AUTO_INCREMENT,
  `item_id` int unsigned NOT NULL,
  `attrname_id` int unsigned NOT NULL,
  PRIMARY KEY (`targetname_id`),
  KEY `item_id` (`item_id`),
  KEY `attrname_id` (`attrname_id`),
  CONSTRAINT `adv_targetname_ibfk_1` FOREIGN KEY (`item_id`) REFERENCES `adv_item` (`item_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `adv_targetname_ibfk_2` FOREIGN KEY (`attrname_id`) REFERENCES `adv_attrname` (`attrname_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv_targetname`
--

LOCK TABLES `adv_targetname` WRITE;
/*!40000 ALTER TABLE `adv_targetname` DISABLE KEYS */;
/*!40000 ALTER TABLE `adv_targetname` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_targetname_insert` AFTER INSERT ON `adv_targetname` FOR EACH ROW BEGIN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
    VALUES (42, NEW.item_id, "targetname", NOW());
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `adv_targetvalue`
--

DROP TABLE IF EXISTS `adv_targetvalue`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `adv_targetvalue` (
  `targetvalue_id` int unsigned NOT NULL AUTO_INCREMENT,
  `targetname_id` int unsigned NOT NULL,
  `value_id` int unsigned NOT NULL,
  PRIMARY KEY (`targetvalue_id`),
  KEY `targetname_id` (`targetname_id`),
  CONSTRAINT `adv_targetvalue_ibfk_1` FOREIGN KEY (`targetname_id`) REFERENCES `adv_targetname` (`targetname_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adv_targetvalue`
--

LOCK TABLES `adv_targetvalue` WRITE;
/*!40000 ALTER TABLE `adv_targetvalue` DISABLE KEYS */;
/*!40000 ALTER TABLE `adv_targetvalue` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `agent`
--

DROP TABLE IF EXISTS `agent`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `agent` (
  `agent_id` int unsigned NOT NULL AUTO_INCREMENT,
  `login` varchar(255) NOT NULL,
  `passwd` varchar(255) NOT NULL,
  `address_id` int unsigned DEFAULT NULL,
  `level` tinyint unsigned NOT NULL DEFAULT '1',
  `notes` text,
  `active` enum('Yes','No','Pause') DEFAULT 'No',
  `created` datetime NOT NULL,
  PRIMARY KEY (`agent_id`),
  UNIQUE KEY `login` (`login`(8)),
  KEY `address_id` (`address_id`),
  CONSTRAINT `agent_ibfk_1` FOREIGN KEY (`address_id`) REFERENCES `add_address` (`address_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `agent`
--

LOCK TABLES `agent` WRITE;
/*!40000 ALTER TABLE `agent` DISABLE KEYS */;
/*!40000 ALTER TABLE `agent` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ch_ac`
--

DROP TABLE IF EXISTS `ch_ac`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ch_ac` (
  `chac_id` int unsigned NOT NULL AUTO_INCREMENT,
  `entitytype_id` tinyint unsigned NOT NULL,
  `entity_id` int unsigned NOT NULL,
  `channel_id` smallint unsigned NOT NULL,
  PRIMARY KEY (`chac_id`),
  UNIQUE KEY `channel` (`entitytype_id`,`entity_id`,`channel_id`),
  KEY `channel_id` (`channel_id`),
  CONSTRAINT `ch_ac_ibfk_1` FOREIGN KEY (`entitytype_id`) REFERENCES `def_entitytype` (`entitytype_id`) ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT `ch_ac_ibfk_2` FOREIGN KEY (`channel_id`) REFERENCES `def_channel` (`channel_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ch_ac`
--

LOCK TABLES `ch_ac` WRITE;
/*!40000 ALTER TABLE `ch_ac` DISABLE KEYS */;
INSERT INTO `ch_ac` VALUES (5,31,51,7),(6,31,51,8);
/*!40000 ALTER TABLE `ch_ac` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_chac_insert` AFTER INSERT ON `ch_ac` FOR EACH ROW BEGIN
	INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
	VALUES (NEW.entitytype_id, NEW.entity_id, "ch_ac", NOW());
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_chac_delete` AFTER DELETE ON `ch_ac` FOR EACH ROW BEGIN
	INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
	VALUES (OLD.entitytype_id, OLD.entity_id, "ch_ac_", NOW());
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `ch_belong`
--

DROP TABLE IF EXISTS `ch_belong`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ch_belong` (
  `chbelong_id` int unsigned NOT NULL AUTO_INCREMENT,
  `entitytype_id` tinyint unsigned NOT NULL,
  `entity_id` int unsigned NOT NULL,
  `channel_id` smallint unsigned NOT NULL,
  PRIMARY KEY (`chbelong_id`),
  UNIQUE KEY `channel` (`entitytype_id`,`entity_id`,`channel_id`),
  KEY `channel_id` (`channel_id`),
  CONSTRAINT `ch_belong_ibfk_1` FOREIGN KEY (`entitytype_id`) REFERENCES `def_entitytype` (`entitytype_id`) ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT `ch_belong_ibfk_2` FOREIGN KEY (`channel_id`) REFERENCES `def_channel` (`channel_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ch_belong`
--

LOCK TABLES `ch_belong` WRITE;
/*!40000 ALTER TABLE `ch_belong` DISABLE KEYS */;
/*!40000 ALTER TABLE `ch_belong` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_belong_insert` AFTER INSERT ON `ch_belong` FOR EACH ROW BEGIN
	INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
	VALUES (NEW.entitytype_id, NEW.entity_id, "ch_belong", NOW());
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_belong_delete` AFTER DELETE ON `ch_belong` FOR EACH ROW BEGIN
	INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created)
	VALUES (OLD.entitytype_id, OLD.entity_id, "ch_belong_", NOW());
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `cron_halfhour`
--

DROP TABLE IF EXISTS `cron_halfhour`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cron_halfhour` (
  `halfhour_id` int unsigned NOT NULL AUTO_INCREMENT,
  `entitytype_id` tinyint unsigned NOT NULL,
  `entity_id` int unsigned NOT NULL,
  `status` enum('new','processing','done') DEFAULT 'new',
  `why` enum('Yes','No','New','Pause','Pass2','Prepare','targetname','bw','channel','creative','creative_','ac','ac_','ch_ac','ch_ac_','ch_belong','ch_belong_','landing','content') DEFAULT 'Yes',
  `created` datetime NOT NULL,
  PRIMARY KEY (`halfhour_id`),
  KEY `status` (`status`),
  KEY `entitytype_id` (`entitytype_id`,`entity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cron_halfhour`
--

LOCK TABLES `cron_halfhour` WRITE;
/*!40000 ALTER TABLE `cron_halfhour` DISABLE KEYS */;
/*!40000 ALTER TABLE `cron_halfhour` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `daily_adv`
--

DROP TABLE IF EXISTS `daily_adv`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `daily_adv` (
  `la_id` int unsigned NOT NULL AUTO_INCREMENT,
  `log_id` int unsigned NOT NULL,
  `item_id` int unsigned NOT NULL,
  `campaign_id` int unsigned NOT NULL,
  `adv_id` int unsigned NOT NULL,
  `spend` float DEFAULT NULL,
  `imps` int unsigned DEFAULT NULL,
  `clis` int unsigned DEFAULT NULL,
  PRIMARY KEY (`la_id`),
  KEY `log_id` (`log_id`),
  CONSTRAINT `daily_adv_ibfk_1` FOREIGN KEY (`log_id`) REFERENCES `daily_log` (`log_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `daily_adv`
--

LOCK TABLES `daily_adv` WRITE;
/*!40000 ALTER TABLE `daily_adv` DISABLE KEYS */;
/*!40000 ALTER TABLE `daily_adv` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `daily_log`
--

DROP TABLE IF EXISTS `daily_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `daily_log` (
  `log_id` int unsigned NOT NULL AUTO_INCREMENT,
  `daily` date NOT NULL,
  `spend` float DEFAULT NULL,
  `imps` int unsigned DEFAULT NULL,
  `clis` int unsigned DEFAULT NULL,
  `created` datetime NOT NULL,
  PRIMARY KEY (`log_id`),
  KEY `daily` (`daily`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `daily_log`
--

LOCK TABLES `daily_log` WRITE;
/*!40000 ALTER TABLE `daily_log` DISABLE KEYS */;
/*!40000 ALTER TABLE `daily_log` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `daily_pub`
--

DROP TABLE IF EXISTS `daily_pub`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `daily_pub` (
  `lp_id` int unsigned NOT NULL AUTO_INCREMENT,
  `log_id` int unsigned DEFAULT NULL,
  `slot_id` int unsigned NOT NULL,
  `site_id` int unsigned NOT NULL,
  `pub_id` int unsigned NOT NULL,
  `spend` float DEFAULT NULL,
  `imps` int unsigned DEFAULT NULL,
  `clis` int unsigned DEFAULT NULL,
  PRIMARY KEY (`lp_id`),
  KEY `log_id` (`log_id`),
  CONSTRAINT `daily_pub_ibfk_1` FOREIGN KEY (`log_id`) REFERENCES `daily_log` (`log_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `daily_pub`
--

LOCK TABLES `daily_pub` WRITE;
/*!40000 ALTER TABLE `daily_pub` DISABLE KEYS */;
/*!40000 ALTER TABLE `daily_pub` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `daily_pub_adv`
--

DROP TABLE IF EXISTS `daily_pub_adv`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `daily_pub_adv` (
  `lpa_id` int unsigned NOT NULL AUTO_INCREMENT,
  `lp_id` int unsigned NOT NULL,
  `la_id` int unsigned NOT NULL,
  `spend` float DEFAULT NULL,
  `imps` int unsigned DEFAULT NULL,
  `clis` int unsigned DEFAULT NULL,
  PRIMARY KEY (`lpa_id`),
  KEY `lp_id` (`lp_id`),
  KEY `la_id` (`la_id`),
  CONSTRAINT `daily_pub_adv_ibfk_1` FOREIGN KEY (`lp_id`) REFERENCES `daily_pub` (`lp_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `daily_pub_adv_ibfk_2` FOREIGN KEY (`la_id`) REFERENCES `daily_adv` (`la_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `daily_pub_adv`
--

LOCK TABLES `daily_pub_adv` WRITE;
/*!40000 ALTER TABLE `daily_pub_adv` DISABLE KEYS */;
/*!40000 ALTER TABLE `daily_pub_adv` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `def_channel`
--

DROP TABLE IF EXISTS `def_channel`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `def_channel` (
  `channel_id` smallint unsigned NOT NULL,
  `channel_name` varchar(255) NOT NULL,
  `level` tinyint unsigned DEFAULT '0',
  `parent` smallint unsigned DEFAULT NULL,
  `full_name` varchar(255) NOT NULL,
  PRIMARY KEY (`channel_id`),
  KEY `parent` (`parent`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `def_channel`
--

LOCK TABLES `def_channel` WRITE;
/*!40000 ALTER TABLE `def_channel` DISABLE KEYS */;
/*!40000 ALTER TABLE `def_channel` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `def_city`
--

DROP TABLE IF EXISTS `def_city`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `def_city` (
  `city_id` int unsigned NOT NULL,
  `city_name` varchar(255) NOT NULL,
  `state_id` int unsigned NOT NULL,
  `postal` varchar(255) DEFAULT NULL,
  `latitude` double DEFAULT NULL,
  `longitude` double DEFAULT NULL,
  PRIMARY KEY (`city_id`),
  KEY `state_id` (`state_id`),
  CONSTRAINT `def_city_ibfk_1` FOREIGN KEY (`state_id`) REFERENCES `def_state` (`state_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `def_city`
--

LOCK TABLES `def_city` WRITE;
/*!40000 ALTER TABLE `def_city` DISABLE KEYS */;
/*!40000 ALTER TABLE `def_city` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `def_continent`
--

DROP TABLE IF EXISTS `def_continent`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `def_continent` (
  `continent_id` tinyint unsigned NOT NULL,
  `continent_code` char(2) NOT NULL,
  PRIMARY KEY (`continent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `def_continent`
--

LOCK TABLES `def_continent` WRITE;
/*!40000 ALTER TABLE `def_continent` DISABLE KEYS */;
INSERT INTO `def_continent` VALUES (0,'00'),(1,'AF'),(2,'AN'),(3,'AS'),(4,'EU'),(5,'NA'),(6,'OC'),(7,'SA');
/*!40000 ALTER TABLE `def_continent` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `def_country`
--

DROP TABLE IF EXISTS `def_country`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `def_country` (
  `country_id` int unsigned NOT NULL,
  `autoid` tinyint unsigned auto_increment not null,
  `continent_id` tinyint unsigned DEFAULT NULL,
  `country_code` char(2) NOT NULL,
  `country_name` varchar(255) Not NULL,
  `is_euro` enum('Yes','No') DEFAULT 'No',
  `alpha3` char(3),
  `numeric_code` char(3),
  locale_code char(5),
  `active` enum('Yes','No') DEFAULT 'No',
  PRIMARY KEY (`country_id`),
  UNIQUE KEY `country_code` (`country_code`),
  KEY `continent_id` (`continent_id`),
  index `autoid` (`autoid`),
  CONSTRAINT `def_country_ibfk_1` FOREIGN KEY (`continent_id`) REFERENCES `def_continent` (`continent_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `def_country`
--

LOCK TABLES `def_country` WRITE;
/*!40000 ALTER TABLE `def_country` DISABLE KEYS */;
/*!40000 ALTER TABLE `def_country` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `def_dma`
--

DROP TABLE IF EXISTS `def_dma`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `def_dma` (
  `dma_id` smallint unsigned NOT NULL AUTO_INCREMENT,
  `city_id` int unsigned NOT NULL,
  `metro_code` varchar(255) NOT NULL,
  `description` varchar(255) NOT NULL,
  PRIMARY KEY (`dma_id`),
  KEY `city_id` (`city_id`),
  CONSTRAINT `def_dma_ibfk_1` FOREIGN KEY (`city_id`) REFERENCES `def_city` (`city_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `def_dma`
--

LOCK TABLES `def_dma` WRITE;
/*!40000 ALTER TABLE `def_dma` DISABLE KEYS */;
/*!40000 ALTER TABLE `def_dma` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `def_entitytype`
--

DROP TABLE IF EXISTS `def_entitytype`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `def_entitytype` (
  `entitytype_id` tinyint unsigned NOT NULL,
  `table_name` varchar(255) NOT NULL,
  `id_name` varchar(255) NOT NULL,
  PRIMARY KEY (`entitytype_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `def_entitytype`
--

LOCK TABLES `def_entitytype` WRITE;
/*!40000 ALTER TABLE `def_entitytype` DISABLE KEYS */;
INSERT INTO `def_entitytype` VALUES (1,'admin','admin_id'),(3,'pub','pub_id'),(4,'adv','adv_id'),(5,'anon','anon_id'),(31,'pub_site','site_id'),(32,'pub_slot','slot_id'),(41,'adv_campaign','campaign_id'),(42,'adv_item','item_id');
/*!40000 ALTER TABLE `def_entitytype` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `def_isp`
--

DROP TABLE IF EXISTS `def_isp`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `def_isp` (
  `isp_id` smallint unsigned NOT NULL AUTO_INCREMENT,
  `isp_name` varchar(255) NOT NULL,
  `counts` int DEFAULT '0',
  PRIMARY KEY (`isp_id`),
  KEY `isp_name` (`isp_name`(8))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `def_isp`
--

LOCK TABLES `def_isp` WRITE;
/*!40000 ALTER TABLE `def_isp` DISABLE KEYS */;
/*!40000 ALTER TABLE `def_isp` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `def_paytype`
--

DROP TABLE IF EXISTS `def_paytype`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `def_paytype` (
  `paytype_id` tinyint unsigned NOT NULL,
  `paytype_value` varchar(255) NOT NULL,
  PRIMARY KEY (`paytype_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `def_paytype`
--

LOCK TABLES `def_paytype` WRITE;
/*!40000 ALTER TABLE `def_paytype` DISABLE KEYS */;
INSERT INTO `def_paytype` VALUES (1,'Cash'),(2,'Debt'),(3,'CC'),(4,'Cheque'),(5,'WeChat'),(6,'Alipay');
/*!40000 ALTER TABLE `def_paytype` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `def_size`
--

DROP TABLE IF EXISTS `def_size`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `def_size` (
  `size_id` int unsigned NOT NULL AUTO_INCREMENT,
  `size_name` varchar(255) NOT NULL,
  `layout` enum('Horizontal','Button','Vertical','Full','Square') NOT NULL,
  `width` smallint unsigned NOT NULL,
  `height` smallint unsigned NOT NULL,
  PRIMARY KEY (`size_id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `def_size`
--

LOCK TABLES `def_size` WRITE;
/*!40000 ALTER TABLE `def_size` DISABLE KEYS */;
INSERT INTO `def_size` VALUES (1,'Half Banner','Horizontal',234,60),(2,'Banner','Horizontal',468,60),(3,'Leaderboard','Horizontal',728,90),(4,'Micro Bar','Button',88,31),(5,'Button 2','Button',120,60),(6,'Button 1','Button',120,90),(7,'Button','Button',125,125),(8,'Vertical Banner','Vertical',120,240),(9,'Skyscraper','Vertical',120,600),(10,'Wide Skyscraper','Vertical',160,600),(11,'Vertical Rectangle','Vertical',240,400),(12,'Small Rectangle','Square',180,150),(13,'Small Square','Square',200,200),(14,'Square','Square',250,250),(15,'3:1 Rectangle','Square',300,100),(16,'Medium Rectangle','Square',300,250),(17,'Large Rectangle','Square',336,280),(18,'Half Page Ad','Full',300,600);
/*!40000 ALTER TABLE `def_size` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `def_state`
--

DROP TABLE IF EXISTS `def_state`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `def_state` (
  `state_id` int unsigned AUTO_INCREMENT NOT NULL,
  `country_id` int unsigned NOT NULL,
  `state_code` char(4) DEFAULT NULL,
  `state_name` varchar(255) DEFAULT NULL,
  `english_name` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`state_id`),
  KEY `country_id` (`country_id`),
  CONSTRAINT `def_state_ibfk_1` FOREIGN KEY (`country_id`) REFERENCES `def_country` (`country_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `def_state`
--

LOCK TABLES `def_state` WRITE;
/*!40000 ALTER TABLE `def_state` DISABLE KEYS */;
/*!40000 ALTER TABLE `def_state` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `his_balance`
--

DROP TABLE IF EXISTS `his_balance`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `his_balance` (
  `his_balance_id` int unsigned NOT NULL AUTO_INCREMENT,
  `balance_id` int unsigned NOT NULL,
  `budget_old` float NOT NULL,
  `budget_add` float NOT NULL,
  `budget_new` float NOT NULL,
  `created` datetime DEFAULT NULL,
  PRIMARY KEY (`his_balance_id`),
  KEY `balance_id` (`balance_id`),
  CONSTRAINT `his_balance_ibfk_1` FOREIGN KEY (`balance_id`) REFERENCES `adv_balance` (`balance_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `his_balance`
--

LOCK TABLES `his_balance` WRITE;
/*!40000 ALTER TABLE `his_balance` DISABLE KEYS */;
/*!40000 ALTER TABLE `his_balance` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `his_payment`
--

DROP TABLE IF EXISTS `his_payment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `his_payment` (
  `his_payment_id` int unsigned NOT NULL AUTO_INCREMENT,
  `payment_id` int unsigned NOT NULL,
  `amount` float NOT NULL,
  `balance_new` float NOT NULL,
  `balance_old` float NOT NULL DEFAULT '0',
  `created` datetime NOT NULL,
  PRIMARY KEY (`his_payment_id`),
  KEY `payment_id` (`payment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `his_payment`
--

LOCK TABLES `his_payment` WRITE;
/*!40000 ALTER TABLE `his_payment` DISABLE KEYS */;
/*!40000 ALTER TABLE `his_payment` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ledger_adv`
--

DROP TABLE IF EXISTS `ledger_adv`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ledger_adv` (
  `la_id` int unsigned NOT NULL AUTO_INCREMENT,
  `log_id` int unsigned NOT NULL,
  `item_id` int unsigned NOT NULL,
  `campaign_id` int unsigned NOT NULL,
  `adv_id` int unsigned NOT NULL,
  `spend` float DEFAULT NULL,
  `imps` int unsigned DEFAULT NULL,
  `clis` int unsigned DEFAULT NULL,
  PRIMARY KEY (`la_id`),
  KEY `log_id` (`log_id`),
  CONSTRAINT `ledger_adv_ibfk_1` FOREIGN KEY (`log_id`) REFERENCES `ledger_log` (`log_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ledger_adv`
--

LOCK TABLES `ledger_adv` WRITE;
/*!40000 ALTER TABLE `ledger_adv` DISABLE KEYS */;
/*!40000 ALTER TABLE `ledger_adv` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ledger_log`
--

DROP TABLE IF EXISTS `ledger_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ledger_log` (
  `log_id` int unsigned NOT NULL AUTO_INCREMENT,
  `timely` datetime NOT NULL,
  `spend` float DEFAULT NULL,
  `imps` int unsigned DEFAULT NULL,
  `clis` int unsigned DEFAULT NULL,
  `created` datetime NOT NULL,
  PRIMARY KEY (`log_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ledger_log`
--

LOCK TABLES `ledger_log` WRITE;
/*!40000 ALTER TABLE `ledger_log` DISABLE KEYS */;
/*!40000 ALTER TABLE `ledger_log` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ledger_pub`
--

DROP TABLE IF EXISTS `ledger_pub`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ledger_pub` (
  `lp_id` int unsigned NOT NULL AUTO_INCREMENT,
  `log_id` int unsigned DEFAULT NULL,
  `slot_id` int unsigned NOT NULL,
  `site_id` int unsigned NOT NULL,
  `pub_id` int unsigned NOT NULL,
  `spend` float DEFAULT NULL,
  `imps` int unsigned DEFAULT NULL,
  `clis` int unsigned DEFAULT NULL,
  PRIMARY KEY (`lp_id`),
  KEY `log_id` (`log_id`),
  CONSTRAINT `ledger_pub_ibfk_1` FOREIGN KEY (`log_id`) REFERENCES `ledger_log` (`log_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ledger_pub`
--

LOCK TABLES `ledger_pub` WRITE;
/*!40000 ALTER TABLE `ledger_pub` DISABLE KEYS */;
/*!40000 ALTER TABLE `ledger_pub` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ledger_pub_adv`
--

DROP TABLE IF EXISTS `ledger_pub_adv`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ledger_pub_adv` (
  `lpa_id` int unsigned NOT NULL AUTO_INCREMENT,
  `lp_id` int unsigned NOT NULL,
  `la_id` int unsigned NOT NULL,
  `spend` float DEFAULT NULL,
  `imps` int unsigned DEFAULT NULL,
  `clis` int unsigned DEFAULT NULL,
  PRIMARY KEY (`lpa_id`),
  KEY `lp_id` (`lp_id`),
  KEY `la_id` (`la_id`),
  CONSTRAINT `ledger_pub_adv_ibfk_1` FOREIGN KEY (`lp_id`) REFERENCES `ledger_pub` (`lp_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `ledger_pub_adv_ibfk_2` FOREIGN KEY (`la_id`) REFERENCES `ledger_adv` (`la_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ledger_pub_adv`
--

LOCK TABLES `ledger_pub_adv` WRITE;
/*!40000 ALTER TABLE `ledger_pub_adv` DISABLE KEYS */;
/*!40000 ALTER TABLE `ledger_pub_adv` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pay_alipay`
--

DROP TABLE IF EXISTS `pay_alipay`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pay_alipay` (
  `alipay_id` int unsigned NOT NULL AUTO_INCREMENT,
  `payment_id` int unsigned NOT NULL,
  `sender_name` varchar(255) DEFAULT NULL,
  `sender_id` varchar(255) DEFAULT NULL,
  `transaction_id` varchar(255) DEFAULT NULL,
  `status` enum('New','Authorized','Expired','Failed') DEFAULT 'New',
  `created` datetime DEFAULT NULL,
  `ip` varchar(15) DEFAULT NULL,
  PRIMARY KEY (`alipay_id`),
  KEY `payment_id` (`payment_id`),
  CONSTRAINT `pay_alipay_ibfk_1` FOREIGN KEY (`payment_id`) REFERENCES `pay_payment` (`payment_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pay_alipay`
--

LOCK TABLES `pay_alipay` WRITE;
/*!40000 ALTER TABLE `pay_alipay` DISABLE KEYS */;
/*!40000 ALTER TABLE `pay_alipay` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pay_cc`
--

DROP TABLE IF EXISTS `pay_cc`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pay_cc` (
  `cc_id` int unsigned NOT NULL AUTO_INCREMENT,
  `payment_id` int unsigned NOT NULL,
  `cardtype` enum('Visa','Master','AmExpress') DEFAULT 'Visa',
  `cardnumber` varchar(255) NOT NULL,
  `expire` date NOT NULL,
  `address_id` int unsigned NOT NULL,
  `transaction_id` varchar(255) DEFAULT NULL,
  `status` enum('New','Authorized','Expired','Failed') DEFAULT 'New',
  `created` datetime DEFAULT NULL,
  `ip` varchar(15) DEFAULT NULL,
  PRIMARY KEY (`cc_id`),
  KEY `address_id` (`address_id`),
  KEY `payment_id` (`payment_id`),
  CONSTRAINT `pay_cc_ibfk_1` FOREIGN KEY (`address_id`) REFERENCES `add_address` (`address_id`) ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT `pay_cc_ibfk_2` FOREIGN KEY (`payment_id`) REFERENCES `pay_payment` (`payment_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pay_cc`
--

LOCK TABLES `pay_cc` WRITE;
/*!40000 ALTER TABLE `pay_cc` DISABLE KEYS */;
/*!40000 ALTER TABLE `pay_cc` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pay_cheque`
--

DROP TABLE IF EXISTS `pay_cheque`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pay_cheque` (
  `cheque_id` int unsigned NOT NULL AUTO_INCREMENT,
  `payment_id` int unsigned NOT NULL,
  `accountype` enum('Checking','Saving') DEFAULT 'Checking',
  `bank` varchar(255) NOT NULL,
  `routing_number` varchar(255) NOT NULL,
  `account_number` varchar(255) NOT NULL,
  `address_id` int unsigned NOT NULL,
  `transaction_id` varchar(255) DEFAULT NULL,
  `status` enum('New','Authorized','Expired','Failed') DEFAULT 'New',
  `created` datetime DEFAULT NULL,
  `ip` varchar(15) DEFAULT NULL,
  PRIMARY KEY (`cheque_id`),
  KEY `address_id` (`address_id`),
  KEY `payment_id` (`payment_id`),
  CONSTRAINT `pay_cheque_ibfk_1` FOREIGN KEY (`address_id`) REFERENCES `add_address` (`address_id`) ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT `pay_cheque_ibfk_2` FOREIGN KEY (`payment_id`) REFERENCES `pay_payment` (`payment_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pay_cheque`
--

LOCK TABLES `pay_cheque` WRITE;
/*!40000 ALTER TABLE `pay_cheque` DISABLE KEYS */;
/*!40000 ALTER TABLE `pay_cheque` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pay_payment`
--

DROP TABLE IF EXISTS `pay_payment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pay_payment` (
  `payment_id` int unsigned NOT NULL AUTO_INCREMENT,
  `paytype_id` tinyint unsigned NOT NULL,
  `adv_id` int unsigned NOT NULL,
  `amount` float NOT NULL,
  `status` enum('New','Confirmed','Completed','Failed') DEFAULT 'New',
  `ip` varchar(15) DEFAULT NULL,
  `created` datetime NOT NULL,
  `updated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`payment_id`),
  KEY `adv_id` (`adv_id`),
  KEY `paytype_id` (`paytype_id`),
  CONSTRAINT `pay_payment_ibfk_1` FOREIGN KEY (`adv_id`) REFERENCES `adv` (`adv_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `pay_payment_ibfk_2` FOREIGN KEY (`paytype_id`) REFERENCES `def_paytype` (`paytype_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pay_payment`
--

LOCK TABLES `pay_payment` WRITE;
/*!40000 ALTER TABLE `pay_payment` DISABLE KEYS */;
/*!40000 ALTER TABLE `pay_payment` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_payment` AFTER UPDATE ON `pay_payment` FOR EACH ROW BEGIN
  DECLARE advID int unsigned;
  DECLARE advBalance float;
  IF (OLD.status="New" && NEW.status="Confirmed") THEN
    SELECT adv.adv_id, adv.balance INTO advID, advBalance FROM pay_payment INNER JOIN adv USING (adv_id) WHERE payment_id=New.payment_id;
    UPDATE adv SET balance=advBalance+NEW.amount WHERE adv_id=advID;
    INSERT INTO his_payment (payment_id, amount, balance_new, balance_old, created) VALUES (NEW.payment_id, NEW.amount, advBalance+NEW.amount, advBalance, NOW());
  END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `pay_wechat`
--

DROP TABLE IF EXISTS `pay_wechat`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pay_wechat` (
  `wechat_id` int unsigned NOT NULL AUTO_INCREMENT,
  `payment_id` int unsigned NOT NULL,
  `sender_name` varchar(255) DEFAULT NULL,
  `sender_id` varchar(255) DEFAULT NULL,
  `transaction_id` varchar(255) DEFAULT NULL,
  `status` enum('New','Authorized','Expired','Failed') DEFAULT 'New',
  `created` datetime DEFAULT NULL,
  `ip` varchar(15) DEFAULT NULL,
  PRIMARY KEY (`wechat_id`),
  KEY `payment_id` (`payment_id`),
  CONSTRAINT `pay_wechat_ibfk_1` FOREIGN KEY (`payment_id`) REFERENCES `pay_payment` (`payment_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pay_wechat`
--

LOCK TABLES `pay_wechat` WRITE;
/*!40000 ALTER TABLE `pay_wechat` DISABLE KEYS */;
/*!40000 ALTER TABLE `pay_wechat` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pub`
--

DROP TABLE IF EXISTS `pub`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pub` (
  `pub_id` int unsigned NOT NULL,
  `email` varchar(255) NOT NULL,
  `passwd` char(40) NOT NULL,
  `firstname` varchar(255) DEFAULT NULL,
  `lastname` varchar(255) DEFAULT NULL,
  `timezone_id` tinyint unsigned DEFAULT NULL,
  `address_id` int unsigned DEFAULT NULL,
  `created` datetime DEFAULT NULL,
  `active` enum('Yes','No','New') DEFAULT 'New',
  `access_order` enum('White','Black') DEFAULT 'Black',
  `updated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`pub_id`),
  UNIQUE KEY `email` (`email`(20)),
  KEY `address_id` (`address_id`),
  CONSTRAINT `pub_ibfk_1` FOREIGN KEY (`address_id`) REFERENCES `add_address` (`address_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pub`
--

LOCK TABLES `pub` WRITE;
/*!40000 ALTER TABLE `pub` DISABLE KEYS */;
/*!40000 ALTER TABLE `pub` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_pub` AFTER UPDATE ON `pub` FOR EACH ROW BEGIN

  IF ((NEW.active = "No") && (OLD.active = "Yes")) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created) 
      VALUES (3, NEW.pub_id, NEW.active, NOW());
  END IF;
  IF ((NEW.access_order <=> OLD.access_order) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created) 
      VALUES (3, NEW.pub_id, 'bw', NOW());
  END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `pub_ip`
--

DROP TABLE IF EXISTS `pub_ip`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pub_ip` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `ip` int unsigned NOT NULL,
  `email` varchar(255) NOT NULL,
  `updated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `ret` enum('fail','success') NOT NULL DEFAULT 'fail',
  PRIMARY KEY (`id`),
  KEY `updated` (`updated`),
  KEY `ip` (`ip`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pub_ip`
--

LOCK TABLES `pub_ip` WRITE;
/*!40000 ALTER TABLE `pub_ip` DISABLE KEYS */;
/*!40000 ALTER TABLE `pub_ip` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pub_referer`
--

DROP TABLE IF EXISTS `pub_referer`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pub_referer` (
  `referer_id` int unsigned NOT NULL AUTO_INCREMENT,
  `site_id` int unsigned DEFAULT '0',
  `referer` varchar(255) NOT NULL,
  PRIMARY KEY (`referer_id`),
  KEY `site_id` (`site_id`),
  CONSTRAINT `pub_referer_ibfk_1` FOREIGN KEY (`site_id`) REFERENCES `pub_site` (`site_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pub_referer`
--

LOCK TABLES `pub_referer` WRITE;
/*!40000 ALTER TABLE `pub_referer` DISABLE KEYS */;
/*!40000 ALTER TABLE `pub_referer` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pub_site`
--

DROP TABLE IF EXISTS `pub_site`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pub_site` (
  `site_id` int unsigned NOT NULL AUTO_INCREMENT,
  `pub_id` int unsigned NOT NULL,
  `site_name` varchar(255) NOT NULL,
  `site_url` varchar(255) NOT NULL,
  `access_order` enum('White','Black','Inherit') DEFAULT 'Inherit',
  `active` enum('Yes','No','New') DEFAULT 'Yes',
  `created` datetime DEFAULT NULL,
  PRIMARY KEY (`site_id`),
  KEY `pub_id` (`pub_id`),
  CONSTRAINT `pub_site_ibfk_1` FOREIGN KEY (`pub_id`) REFERENCES `pub` (`pub_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pub_site`
--

LOCK TABLES `pub_site` WRITE;
/*!40000 ALTER TABLE `pub_site` DISABLE KEYS */;
/*!40000 ALTER TABLE `pub_site` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_site` AFTER UPDATE ON `pub_site` FOR EACH ROW BEGIN

  IF ((NEW.active = "No") && (OLD.active = "Yes")) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created) 
      VALUES (31, NEW.site_id, NEW.active, NOW());
  END IF;
  IF ((NEW.access_order <=> OLD.access_order) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created) 
      VALUES (31, NEW.site_id, 'bw', NOW());
  END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `pub_slot`
--

DROP TABLE IF EXISTS `pub_slot`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pub_slot` (
  `slot_id` int unsigned NOT NULL AUTO_INCREMENT,
  `site_id` int unsigned DEFAULT '0',
  `slot_name` varchar(255) NOT NULL,
  `size_id` int unsigned NOT NULL,
  `bidfloor` float DEFAULT '0',

  `qa_slot` int unsigned DEFAULT '0',
  `fl_item` int unsigned DEFAULT '0',
  `qa_language` enum("EN","ES","RU","DE","FR","JA","PT","TR","IT","FA","NL","PL","ZH","VI","ID","CS","KO","UK","AR","EL","FI","HE","SV","RO","HU","TH","DA","SK","BG","SR","NB","Other") DEFAULT 'EN',
  `qa_device` enum("0","1","2","3","4","5","6","7") DEFAULT '0',
  `qa_position` enum('0','1','2','3','4','5''6','7') DEFAULT '0',
  `fl_creative` set('0','1','2','3','4','5','6','7','8','9','10','11','12','13','14','15','16','17','18') DEFAULT '0',
  `fl_expnd` set('0','1','2','3','4','5') DEFAULT '0,1,2,3,4,5',
  `fl_mime` set('0','1','2','3','4') DEFAULT '0,1,2,3,4',

  `channel_order` enum('White','Black') DEFAULT 'Black',
  `active` enum('Yes','No','New') DEFAULT 'Yes',
  `created` datetime DEFAULT NULL,
  PRIMARY KEY (`slot_id`),
  KEY `size_id` (`size_id`),
  KEY `site_id` (`site_id`),
  CONSTRAINT `pub_slot_ibfk_1` FOREIGN KEY (`site_id`) REFERENCES `pub_site` (`site_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pub_slot`
--

LOCK TABLES `pub_slot` WRITE;
/*!40000 ALTER TABLE `pub_slot` DISABLE KEYS */;
/*!40000 ALTER TABLE `pub_slot` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE TRIGGER `trig_slot` AFTER UPDATE ON `pub_slot` FOR EACH ROW BEGIN

	IF ((NEW.active = "No") && (OLD.active = "Yes")) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created) 
      VALUES (32, NEW.slot_id, NEW.active, NOW());
  END IF;
  IF ((NEW.channel_order <=> OLD.channel_order) = 0) THEN
    INSERT INTO cron_halfhour (entitytype_id, entity_id, why, created) 
      VALUES (32, NEW.slot_id, 'channel', NOW());
  END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `pub_weight`
--

DROP TABLE IF EXISTS `pub_weight`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pub_weight` (
  `weight_id` int unsigned NOT NULL AUTO_INCREMENT,
  `slot_id` int unsigned NOT NULL,
  `item_id` int unsigned NOT NULL,
  `weight` float DEFAULT '0',
  `created` datetime DEFAULT NULL,
  PRIMARY KEY (`weight_id`),
  UNIQUE KEY `slotitem` (`slot_id`,`item_id`),
  KEY `item_id` (`item_id`),
  CONSTRAINT `pub_weight_ibfk_1` FOREIGN KEY (`slot_id`) REFERENCES `pub_slot` (`slot_id`) ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT `pub_weight_ibfk_2` FOREIGN KEY (`item_id`) REFERENCES `adv_item` (`item_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pub_weight`
--

LOCK TABLES `pub_weight` WRITE;
/*!40000 ALTER TABLE `pub_weight` DISABLE KEYS */;
/*!40000 ALTER TABLE `pub_weight` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pub_white`
--

DROP TABLE IF EXISTS `pub_white`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pub_white` (
  `white_id` int unsigned NOT NULL AUTO_INCREMENT,
  `slot_id` int unsigned NOT NULL,
  `item_id` int unsigned NOT NULL,
  PRIMARY KEY (`white_id`),
  KEY `other` (`slot_id`,`item_id`),
  KEY `item_id` (`item_id`),
  CONSTRAINT `pub_white_ibfk_1` FOREIGN KEY (`slot_id`) REFERENCES `pub_slot` (`slot_id`) ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT `pub_white_ibfk_2` FOREIGN KEY (`item_id`) REFERENCES `adv_item` (`item_id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pub_white`
--

LOCK TABLES `pub_white` WRITE;
/*!40000 ALTER TABLE `pub_white` DISABLE KEYS */;
/*!40000 ALTER TABLE `pub_white` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Temporary view structure for view `view_payment`
--

DROP TABLE IF EXISTS `view_payment`;
/*!50001 DROP VIEW IF EXISTS `view_payment`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `view_payment` AS SELECT 
 1 AS `payment_id`,
 1 AS `status`,
 1 AS `entitytype_id`,
 1 AS `entity_id`,
 1 AS `adv_id`,
 1 AS `amount`,
 1 AS `ip`,
 1 AS `created`,
 1 AS `paytype_value`,
 1 AS `sender_name`,
 1 AS `sender_id`,
 1 AS `transaction_id`*/;
SET character_set_client = @saved_cs_client;

--
-- Dumping routines for database 'gotest'
--
/*!50003 DROP PROCEDURE IF EXISTS `proc_adv` */;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
CREATE PROCEDURE `proc_adv`(IN i_email VARCHAR(255), IN i_passwd VARCHAR(40), IN i_ip INT unsigned, OUT o_adv_id INT unsigned, OUT o_email VARCHAR(48), OUT o_company VARCHAR(255), OUT o_contact varchar(255), OUT o_timezone_id tinyint unsigned)
BEGIN
  DECLARE c1 INT;
  DECLARE c2 INT;
  SELECT COUNT(*) INTO c1 FROM adv_ip WHERE ret='fail' AND ip=i_ip AND email=i_email AND (UNIX_TIMESTAMP(updated) >= (UNIX_TIMESTAMP(NOW())-3600));
  SELECT COUNT(*) INTO c2 FROM adv_ip WHERE ret='fail' AND ip=i_ip AND (UNIX_TIMESTAMP(updated) >= (UNIX_TIMESTAMP(NOW())-24*3600));
  IF (c1<=5 AND c2<=20)
  THEN
    SELECT p.adv_id, p.email, p.timezone_id, a.company, a.contact
    INTO o_adv_id, o_email, o_timezone_id, o_company, o_contact
    FROM adv p
    LEFT JOIN add_address a USING (address_id)
    WHERE p.email=i_email and p.passwd=SHA1(concat(i_passwd,i_email)) and p.active='Yes';

    IF ISNULL(o_adv_id)
    THEN
      INSERT INTO adv_ip (ip, email, ret) VALUES (i_ip, i_email, 'fail');
    ELSE
      DELETE FROM adv_ip WHERE ret='fail' AND ip=i_ip AND (UNIX_TIMESTAMP(updated) >= (UNIX_TIMESTAMP(NOW())-24*3600));
      INSERT INTO adv_ip (ip, email, ret) VALUES (i_ip, i_email, 'success');
    END IF;
  ELSE
    SELECT '1030' INTO o_email;
  END IF;

END ;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 DROP PROCEDURE IF EXISTS `proc_adv_as` */;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
CREATE PROCEDURE `proc_adv_as`(IN i_email VARCHAR(255), IN i_ip INT unsigned, OUT o_adv_id INT unsigned, OUT o_email VARCHAR(48), OUT o_company VARCHAR(255), OUT o_contact varchar(255), OUT o_timezone_id tinyint unsigned)
BEGIN
    SELECT p.adv_id, p.email, p.timezone_id, a.company, a.contact
    INTO o_adv_id, o_email, o_timezone_id, o_company, o_contact
    FROM adv p
    LEFT JOIN add_address a USING (address_id)
    WHERE p.email=i_email;
END ;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 DROP PROCEDURE IF EXISTS `proc_pub` */;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
CREATE PROCEDURE `proc_pub`(IN i_email VARCHAR(255), IN i_passwd VARCHAR(40), IN i_ip INT unsigned, OUT o_pub_id INT unsigned, OUT o_email VARCHAR(48), OUT o_company VARCHAR(255), OUT o_contact varchar(255), OUT o_timezone_id tinyint unsigned)
BEGIN
  DECLARE c1 INT;
  DECLARE c2 INT;
  SELECT COUNT(*) INTO c1 FROM pub_ip WHERE ret='fail' AND ip=i_ip AND email=i_email AND (UNIX_TIMESTAMP(updated) >= (UNIX_TIMESTAMP(NOW())-3600));
  SELECT COUNT(*) INTO c2 FROM pub_ip WHERE ret='fail' AND ip=i_ip AND (UNIX_TIMESTAMP(updated) >= (UNIX_TIMESTAMP(NOW())-24*3600));
  IF (c1<=5 AND c2<=20)
  THEN
    SELECT p.pub_id, p.email, p.timezone_id, a.company, a.contact
    INTO o_pub_id, o_email, o_timezone_id, o_company, o_contact
    FROM pub p
    LEFT JOIN add_address a USING (address_id)
    WHERE p.email=i_email and p.passwd=SHA1(concat(i_passwd,i_email)) and p.active='Yes';

    IF ISNULL(o_pub_id)
    THEN
      INSERT INTO pub_ip (ip, email, ret) VALUES (i_ip, i_email, 'fail');
    ELSE
      DELETE FROM pub_ip WHERE ret='fail' AND ip=i_ip AND (UNIX_TIMESTAMP(updated) >= (UNIX_TIMESTAMP(NOW())-24*3600));
      INSERT INTO pub_ip (ip, email, ret) VALUES (i_ip, i_email, 'success');
    END IF;
  ELSE
    SELECT '1030' INTO o_email;
  END IF;

END ;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 DROP PROCEDURE IF EXISTS `proc_pub_as` */;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb3_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
CREATE PROCEDURE `proc_pub_as`(IN i_email VARCHAR(255), IN i_ip INT unsigned, OUT o_pub_id INT unsigned, OUT o_email VARCHAR(48), OUT o_company VARCHAR(255), OUT o_contact varchar(255), OUT o_timezone_id tinyint unsigned)
BEGIN
    SELECT p.pub_id, p.email, p.timezone_id, a.company, a.contact
    INTO o_pub_id, o_email, o_timezone_id, o_company, o_contact
    FROM pub p
    LEFT JOIN add_address a USING (address_id)
    WHERE p.email=i_email;
END ;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Final view structure for view `ViewGroupSlot`
--

/*!50001 DROP VIEW IF EXISTS `ViewGroupSlot`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb3 */;
/*!50001 SET character_set_results     = utf8mb3 */;
/*!50001 SET collation_connection      = utf8mb3_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50001 VIEW `ViewGroupSlot` AS select any_value(`p`.`pub_id`) AS `pub_id`,any_value(`s`.`site_id`) AS `site_id`,`t`.`slot_id` AS `slot_id`,any_value(`a`.`adv_id`) AS `adv_id`,any_value(`c`.`campaign_id`) AS `campaign_id`,`i`.`item_id` AS `item_id`,any_value((`i`.`cost_type` + 0)) AS `cost_type`,any_value(`i`.`cost`) AS `cost`,any_value(`i`.`endx`) AS `endx` from (((((((((((`pub_slot` `t` join `pub_site` `s` on((`t`.`site_id` = `s`.`site_id`))) join `pub` `p` on((`s`.`pub_id` = `p`.`pub_id`))) join `adv_item` `i` on((`t`.`size_id` = `i`.`size_id`))) join `adv_campaign` `c` on((`i`.`campaign_id` = `c`.`campaign_id`))) join `adv` `a` on((`c`.`adv_id` = `a`.`adv_id`))) left join `ac` on(((`ac`.`entitytype_id` = 4) and (`ac`.`entity_id` = `a`.`adv_id`) and (((`ac`.`othertype_id` = 3) and (`ac`.`other_id` = `p`.`pub_id`)) or ((`ac`.`othertype_id` = 31) and (`ac`.`other_id` = `s`.`site_id`)))))) left join `ac` `bc` on(((`bc`.`entitytype_id` = 3) and (`bc`.`entity_id` = `p`.`pub_id`) and (((`bc`.`othertype_id` = 4) and (`bc`.`other_id` = `a`.`adv_id`)) or ((`bc`.`othertype_id` = 41) and (`bc`.`other_id` = `c`.`campaign_id`)))))) left join `ch_ac` `ha` on(((`ha`.`entitytype_id` = 42) and (`ha`.`entity_id` = `i`.`item_id`)))) left join `ch_belong` `hb` on(((`hb`.`entitytype_id` = 32) and (`hb`.`entity_id` = `t`.`slot_id`) and (`hb`.`channel_id` = `ha`.`channel_id`)))) left join `ch_ac` `ca` on(((`ca`.`entitytype_id` = 32) and (`ca`.`entity_id` = `t`.`slot_id`)))) left join `ch_belong` `cb` on(((`cb`.`entitytype_id` = 42) and (`cb`.`entity_id` = `i`.`item_id`) and (`ca`.`channel_id` = `cb`.`channel_id`)))) where ((`p`.`active` = 'Yes') and (`s`.`active` = 'Yes') and (`t`.`active` = 'Yes') and (`a`.`active` = 'Yes') and (`c`.`active` = 'Yes') and (`i`.`active` = 'Yes') and (find_in_set(`i`.`qa_mime`,`t`.`fl_mime`) > 0) and (find_in_set(`t`.`qa_language`,`i`.`fl_language`) > 0) and (find_in_set(`t`.`qa_device`,`i`.`fl_device`) > 0) and (find_in_set(`t`.`qa_position`,`i`.`fl_position`) > 0) and (find_in_set(`i`.`qa_expnd`,`t`.`fl_expnd`) > 0) and (find_in_set(`i`.`qa_creative`,`t`.`fl_creative`) > 0) and (((`i`.`qa_item` >> 0) & 7) >= ((`t`.`fl_item` >> 0) & 7)) and (((`i`.`qa_item` >> 3) & 7) >= ((`t`.`fl_item` >> 3) & 7)) and (((`i`.`qa_item` >> 6) & 7) >= ((`t`.`fl_item` >> 6) & 7)) and (((`i`.`qa_item` >> 9) & 7) >= ((`t`.`fl_item` >> 9) & 7)) and (((`i`.`qa_item` >> 12) & 7) >= ((`t`.`fl_item` >> 12) & 7)) and (((`i`.`qa_item` >> 15) & 7) >= ((`t`.`fl_item` >> 15) & 7)) and (((`i`.`fl_slot` >> 0) & 3) <= ((`t`.`qa_slot` >> 0) & 3)) and (((`i`.`fl_slot` >> 2) & 3) <= ((`t`.`qa_slot` >> 2) & 3)) and (((`i`.`fl_slot` >> 4) & 3) <= ((`t`.`qa_slot` >> 4) & 3)) and (((`i`.`fl_slot` >> 6) & 3) <= ((`t`.`qa_slot` >> 6) & 3)) and (((`i`.`fl_slot` >> 8) & 3) <= ((`t`.`qa_slot` >> 8) & 3)) and (((`i`.`fl_slot` >> 10) & 3) <= ((`t`.`qa_slot` >> 10) & 3)) and (((`i`.`fl_slot` >> 12) & 3) <= ((`t`.`qa_slot` >> 12) & 3)) and (((`i`.`fl_slot` >> 14) & 3) <= ((`t`.`qa_slot` >> 14) & 3)) and (((`i`.`fl_slot` >> 16) & 3) <= ((`t`.`qa_slot` >> 16) & 3)) and (((`i`.`fl_slot` >> 18) & 3) <= ((`t`.`qa_slot` >> 18) & 3)) and (((`i`.`fl_slot` >> 20) & 3) <= ((`t`.`qa_slot` >> 20) & 3)) and ((`i`.`endx` >= now()) or (`i`.`endx` is null)) and (((`a`.`access_order` = 'White') and (`ac`.`entity_id` is not null)) or ((`a`.`access_order` = 'Black') and (`ac`.`entity_id` is null))) and (((`i`.`channel_order` = 'Black') and ((`ha`.`entity_id` is null) or (`hb`.`entity_id` is null))) or ((`i`.`channel_order` = 'White') and (`ha`.`entity_id` is not null) and (`hb`.`entity_id` is not null))) and (((`t`.`channel_order` = 'Black') and ((`ca`.`entity_id` is null) or (`cb`.`entity_id` is null))) or ((`t`.`channel_order` = 'White') and (`ca`.`entity_id` is not null) and (`cb`.`entity_id` is not null))) and (((`p`.`access_order` = 'White') and (`bc`.`entity_id` is not null)) or ((`p`.`access_order` = 'Black') and (`bc`.`entity_id` is null)))) group by `t`.`slot_id`,`i`.`item_id` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `ViewSlot`
--

/*!50001 DROP VIEW IF EXISTS `ViewSlot`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb3 */;
/*!50001 SET character_set_results     = utf8mb3 */;
/*!50001 SET collation_connection      = utf8mb3_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50001 VIEW `ViewSlot` AS select `p`.`pub_id` AS `pub_id`,concat(`p`.`firstname`,' ',`p`.`lastname`) AS `pub_name`,`s`.`site_id` AS `site_id`,`s`.`site_name` AS `site_name`,`s`.`site_url` AS `site_url`,`t`.`slot_id` AS `slot_id`,`t`.`slot_name` AS `slot_name`,`i`.`item_id` AS `item_id`,`i`.`item_name` AS `item_name`,`i`.`cost_type` AS `cost_type`,`i`.`cost` AS `cost`,`i`.`startx` AS `startx`,`i`.`endx` AS `endx`,`c`.`campaign_id` AS `campaign_id`,`c`.`campaign_name` AS `campaign_name`,`a`.`adv_id` AS `adv_id`,concat(`a`.`firstname`,' ',`a`.`lastname`) AS `adv_name`,`bc`.`ac_id` AS `ac_id`,`bc`.`entitytype_id` AS `entitytype_id`,`bc`.`entity_id` AS `entity_id`,`bc`.`othertype_id` AS `othertype_id`,`bc`.`other_id` AS `other_id` from (((((((((((`pub_slot` `t` join `pub_site` `s` on((`t`.`site_id` = `s`.`site_id`))) join `pub` `p` on((`s`.`pub_id` = `p`.`pub_id`))) join `adv_item` `i` on((`t`.`size_id` = `i`.`size_id`))) join `adv_campaign` `c` on((`i`.`campaign_id` = `c`.`campaign_id`))) join `adv` `a` on((`c`.`adv_id` = `a`.`adv_id`))) left join `ac` on(((`ac`.`entitytype_id` = 4) and (`ac`.`entity_id` = `a`.`adv_id`) and (((`ac`.`othertype_id` = 3) and (`ac`.`other_id` = `p`.`pub_id`)) or ((`ac`.`othertype_id` = 31) and (`ac`.`other_id` = `s`.`site_id`)))))) left join `ac` `bc` on(((`bc`.`entitytype_id` = 3) and (`bc`.`entity_id` = `p`.`pub_id`) and (((`bc`.`othertype_id` = 4) and (`bc`.`other_id` = `a`.`adv_id`)) or ((`bc`.`othertype_id` = 41) and (`bc`.`other_id` = `c`.`campaign_id`)))))) left join `ch_ac` `ha` on(((`ha`.`entitytype_id` = 42) and (`ha`.`entity_id` = `i`.`item_id`)))) left join `ch_belong` `hb` on(((`hb`.`entitytype_id` = 32) and (`hb`.`entity_id` = `t`.`slot_id`) and (`hb`.`channel_id` = `ha`.`channel_id`)))) left join `ch_ac` `ca` on(((`ca`.`entitytype_id` = 32) and (`ca`.`entity_id` = `t`.`slot_id`)))) left join `ch_belong` `cb` on(((`cb`.`entitytype_id` = 42) and (`cb`.`entity_id` = `i`.`item_id`) and (`ca`.`channel_id` = `cb`.`channel_id`)))) where ((`p`.`active` = 'Yes') and (`s`.`active` = 'Yes') and (`t`.`active` = 'Yes') and (`a`.`active` = 'Yes') and (`c`.`active` = 'Yes') and (`i`.`active` = 'Yes') and (find_in_set(`i`.`qa_mime`,`t`.`fl_mime`) > 0) and (find_in_set(`t`.`qa_language`,`i`.`fl_language`) > 0) and (find_in_set(`t`.`qa_device`,`i`.`fl_device`) > 0) and (find_in_set(`t`.`qa_position`,`i`.`fl_position`) > 0) and (find_in_set(`i`.`qa_expnd`,`t`.`fl_expnd`) > 0) and (find_in_set(`i`.`qa_creative`,`t`.`fl_creative`) > 0) and (((`i`.`qa_item` >> 0) & 7) >= ((`t`.`fl_item` >> 0) & 7)) and (((`i`.`qa_item` >> 3) & 7) >= ((`t`.`fl_item` >> 3) & 7)) and (((`i`.`qa_item` >> 6) & 7) >= ((`t`.`fl_item` >> 6) & 7)) and (((`i`.`qa_item` >> 9) & 7) >= ((`t`.`fl_item` >> 9) & 7)) and (((`i`.`qa_item` >> 12) & 7) >= ((`t`.`fl_item` >> 12) & 7)) and (((`i`.`qa_item` >> 15) & 7) >= ((`t`.`fl_item` >> 15) & 7)) and (((`i`.`fl_slot` >> 0) & 3) <= ((`t`.`qa_slot` >> 0) & 3)) and (((`i`.`fl_slot` >> 2) & 3) <= ((`t`.`qa_slot` >> 2) & 3)) and (((`i`.`fl_slot` >> 4) & 3) <= ((`t`.`qa_slot` >> 4) & 3)) and (((`i`.`fl_slot` >> 6) & 3) <= ((`t`.`qa_slot` >> 6) & 3)) and (((`i`.`fl_slot` >> 8) & 3) <= ((`t`.`qa_slot` >> 8) & 3)) and (((`i`.`fl_slot` >> 10) & 3) <= ((`t`.`qa_slot` >> 10) & 3)) and (((`i`.`fl_slot` >> 12) & 3) <= ((`t`.`qa_slot` >> 12) & 3)) and (((`i`.`fl_slot` >> 14) & 3) <= ((`t`.`qa_slot` >> 14) & 3)) and (((`i`.`fl_slot` >> 16) & 3) <= ((`t`.`qa_slot` >> 16) & 3)) and (((`i`.`fl_slot` >> 18) & 3) <= ((`t`.`qa_slot` >> 18) & 3)) and (((`i`.`fl_slot` >> 20) & 3) <= ((`t`.`qa_slot` >> 20) & 3)) and ((`i`.`endx` >= now()) or (`i`.`endx` is null)) and (((`a`.`access_order` = 'White') and (`ac`.`entity_id` is not null)) or ((`a`.`access_order` = 'Black') and (`ac`.`entity_id` is null))) and (((`i`.`channel_order` = 'Black') and ((`ha`.`entity_id` is null) or (`hb`.`entity_id` is null))) or ((`i`.`channel_order` = 'White') and (`ha`.`entity_id` is not null) and (`hb`.`entity_id` is not null))) and (((`t`.`channel_order` = 'Black') and ((`ca`.`entity_id` is null) or (`cb`.`entity_id` is null))) or ((`t`.`channel_order` = 'White') and (`ca`.`entity_id` is not null) and (`cb`.`entity_id` is not null))) and (((`p`.`access_order` = 'White') and (`bc`.`entity_id` is not null)) or ((`p`.`access_order` = 'Black') and (`bc`.`entity_id` is null)))) */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `ViewSlotOpen`
--

/*!50001 DROP VIEW IF EXISTS `ViewSlotOpen`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb3 */;
/*!50001 SET character_set_results     = utf8mb3 */;
/*!50001 SET collation_connection      = utf8mb3_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50001 VIEW `ViewSlotOpen` AS select `p`.`pub_id` AS `pub_id`,concat(`p`.`firstname`,' ',`p`.`lastname`) AS `pub_name`,`s`.`site_id` AS `site_id`,`s`.`site_name` AS `site_name`,`s`.`site_url` AS `site_url`,`t`.`slot_id` AS `slot_id`,`t`.`slot_name` AS `slot_name`,`i`.`item_id` AS `item_id`,`i`.`item_name` AS `item_name`,`i`.`cost_type` AS `cost_type`,`i`.`cost` AS `cost`,`i`.`startx` AS `startx`,`i`.`endx` AS `endx`,`c`.`campaign_id` AS `campaign_id`,`c`.`campaign_name` AS `campaign_name`,`a`.`adv_id` AS `adv_id`,concat(`a`.`firstname`,' ',`a`.`lastname`) AS `adv_name`,`bc`.`ac_id` AS `ac_id`,`bc`.`entitytype_id` AS `entitytype_id`,`bc`.`entity_id` AS `entity_id`,`bc`.`othertype_id` AS `othertype_id`,`bc`.`other_id` AS `other_id` from (((((((((((`pub_slot` `t` join `pub_site` `s` on((`t`.`site_id` = `s`.`site_id`))) join `pub` `p` on((`s`.`pub_id` = `p`.`pub_id`))) join `adv_item` `i` on((`t`.`size_id` = `i`.`size_id`))) join `adv_campaign` `c` on((`i`.`campaign_id` = `c`.`campaign_id`))) join `adv` `a` on((`c`.`adv_id` = `a`.`adv_id`))) left join `ac` on(((`ac`.`entitytype_id` = 4) and (`ac`.`entity_id` = `a`.`adv_id`) and (((`ac`.`othertype_id` = 3) and (`ac`.`other_id` = `p`.`pub_id`)) or ((`ac`.`othertype_id` = 31) and (`ac`.`other_id` = `s`.`site_id`)))))) left join `ac` `bc` on(((`bc`.`entitytype_id` = 3) and (`bc`.`entity_id` = `p`.`pub_id`) and (((`bc`.`othertype_id` = 4) and (`bc`.`other_id` = `a`.`adv_id`)) or ((`bc`.`othertype_id` = 41) and (`bc`.`other_id` = `c`.`campaign_id`)))))) left join `ch_ac` `ha` on(((`ha`.`entitytype_id` = 42) and (`ha`.`entity_id` = `i`.`item_id`)))) left join `ch_belong` `hb` on(((`hb`.`entitytype_id` = 32) and (`hb`.`entity_id` = `t`.`slot_id`) and (`hb`.`channel_id` = `ha`.`channel_id`)))) left join `ch_ac` `ca` on(((`ca`.`entitytype_id` = 32) and (`ca`.`entity_id` = `t`.`slot_id`)))) left join `ch_belong` `cb` on(((`cb`.`entitytype_id` = 42) and (`cb`.`entity_id` = `i`.`item_id`) and (`ca`.`channel_id` = `cb`.`channel_id`)))) where ((`p`.`active` = 'Yes') and (`s`.`active` = 'Yes') and (`t`.`active` = 'Yes') and (`a`.`active` = 'Yes') and (`c`.`active` = 'Yes') and (`i`.`active` = 'Yes') and (find_in_set(`i`.`qa_mime`,`t`.`fl_mime`) > 0) and (find_in_set(`t`.`qa_language`,`i`.`fl_language`) > 0) and (find_in_set(`t`.`qa_device`,`i`.`fl_device`) > 0) and (find_in_set(`t`.`qa_position`,`i`.`fl_position`) > 0) and (find_in_set(`i`.`qa_expnd`,`t`.`fl_expnd`) > 0) and (find_in_set(`i`.`qa_creative`,`t`.`fl_creative`) > 0) and (((`i`.`qa_item` >> 0) & 7) >= ((`t`.`fl_item` >> 0) & 7)) and (((`i`.`qa_item` >> 3) & 7) >= ((`t`.`fl_item` >> 3) & 7)) and (((`i`.`qa_item` >> 6) & 7) >= ((`t`.`fl_item` >> 6) & 7)) and (((`i`.`qa_item` >> 9) & 7) >= ((`t`.`fl_item` >> 9) & 7)) and (((`i`.`qa_item` >> 12) & 7) >= ((`t`.`fl_item` >> 12) & 7)) and (((`i`.`qa_item` >> 15) & 7) >= ((`t`.`fl_item` >> 15) & 7)) and (((`i`.`fl_slot` >> 0) & 3) <= ((`t`.`qa_slot` >> 0) & 3)) and (((`i`.`fl_slot` >> 2) & 3) <= ((`t`.`qa_slot` >> 2) & 3)) and (((`i`.`fl_slot` >> 4) & 3) <= ((`t`.`qa_slot` >> 4) & 3)) and (((`i`.`fl_slot` >> 6) & 3) <= ((`t`.`qa_slot` >> 6) & 3)) and (((`i`.`fl_slot` >> 8) & 3) <= ((`t`.`qa_slot` >> 8) & 3)) and (((`i`.`fl_slot` >> 10) & 3) <= ((`t`.`qa_slot` >> 10) & 3)) and (((`i`.`fl_slot` >> 12) & 3) <= ((`t`.`qa_slot` >> 12) & 3)) and (((`i`.`fl_slot` >> 14) & 3) <= ((`t`.`qa_slot` >> 14) & 3)) and (((`i`.`fl_slot` >> 16) & 3) <= ((`t`.`qa_slot` >> 16) & 3)) and (((`i`.`fl_slot` >> 18) & 3) <= ((`t`.`qa_slot` >> 18) & 3)) and (((`i`.`fl_slot` >> 20) & 3) <= ((`t`.`qa_slot` >> 20) & 3)) and ((`i`.`endx` >= now()) or (`i`.`endx` is null)) and (((`a`.`access_order` = 'White') and (`ac`.`entity_id` is not null)) or ((`a`.`access_order` = 'Black') and (`ac`.`entity_id` is null))) and (((`i`.`channel_order` = 'Black') and ((`ha`.`entity_id` is null) or (`hb`.`entity_id` is null))) or ((`i`.`channel_order` = 'White') and (`ha`.`entity_id` is not null) and (`hb`.`entity_id` is not null))) and (((`t`.`channel_order` = 'Black') and ((`ca`.`entity_id` is null) or (`cb`.`entity_id` is null))) or ((`t`.`channel_order` = 'White') and (`ca`.`entity_id` is not null) and (`cb`.`entity_id` is not null)))) */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `ViewTaoSlot`
--

/*!50001 DROP VIEW IF EXISTS `ViewTaoSlot`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb3 */;
/*!50001 SET character_set_results     = utf8mb3 */;
/*!50001 SET collation_connection      = utf8mb3_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50001 VIEW `ViewTaoSlot` AS select `t`.`slot_id` AS `slot_id`,`a`.`adv_id` AS `adv_id`,`c`.`campaign_id` AS `campaign_id`,`i`.`item_id` AS `item_id`,(`i`.`cost_type` + 0) AS `cost_type`,`i`.`cost` AS `price`,`i`.`endx` AS `endx`,`i`.`cpm_fc` AS `cpm_fc`,`i`.`cpm_length` AS `cpm_length`,`i`.`cpm_throttle` AS `cpm_throttle`,`i`.`cpc_fc` AS `cpc_fc`,`i`.`cpc_length` AS `cpc_length` from (((((((((((`pub_slot` `t` join `pub_site` `s` on((`t`.`site_id` = `s`.`site_id`))) join `pub` `p` on((`s`.`pub_id` = `p`.`pub_id`))) join `adv_item` `i` on((`t`.`size_id` = `i`.`size_id`))) join `adv_campaign` `c` on((`i`.`campaign_id` = `c`.`campaign_id`))) join `adv` `a` on((`c`.`adv_id` = `a`.`adv_id`))) left join `ac` on(((`ac`.`entitytype_id` = 4) and (`ac`.`entity_id` = `a`.`adv_id`) and (((`ac`.`othertype_id` = 3) and (`ac`.`other_id` = `p`.`pub_id`)) or ((`ac`.`othertype_id` = 31) and (`ac`.`other_id` = `s`.`site_id`)))))) left join `ac` `bc` on(((`bc`.`entitytype_id` = 3) and (`bc`.`entity_id` = `p`.`pub_id`) and (((`bc`.`othertype_id` = 4) and (`bc`.`other_id` = `a`.`adv_id`)) or ((`bc`.`othertype_id` = 41) and (`bc`.`other_id` = `c`.`campaign_id`)))))) left join `ch_ac` `ha` on(((`ha`.`entitytype_id` = 42) and (`ha`.`entity_id` = `i`.`item_id`)))) left join `ch_belong` `hb` on(((`hb`.`entitytype_id` = 32) and (`hb`.`entity_id` = `t`.`slot_id`) and (`hb`.`channel_id` = `ha`.`channel_id`)))) left join `ch_ac` `ca` on(((`ca`.`entitytype_id` = 32) and (`ca`.`entity_id` = `t`.`slot_id`)))) left join `ch_belong` `cb` on(((`cb`.`entitytype_id` = 42) and (`cb`.`entity_id` = `i`.`item_id`) and (`ca`.`channel_id` = `cb`.`channel_id`)))) where ((`p`.`active` = 'Yes') and (`s`.`active` = 'Yes') and (`t`.`active` = 'Yes') and (`a`.`active` = 'Yes') and (`c`.`active` = 'Yes') and (`i`.`active` = 'Yes') and (find_in_set(`i`.`qa_mime`,`t`.`fl_mime`) > 0) and (find_in_set(`t`.`qa_language`,`i`.`fl_language`) > 0) and (find_in_set(`t`.`qa_device`,`i`.`fl_device`) > 0) and (find_in_set(`t`.`qa_position`,`i`.`fl_position`) > 0) and (find_in_set(`i`.`qa_expnd`,`t`.`fl_expnd`) > 0) and (find_in_set(`i`.`qa_creative`,`t`.`fl_creative`) > 0) and (((`i`.`qa_item` >> 0) & 7) >= ((`t`.`fl_item` >> 0) & 7)) and (((`i`.`qa_item` >> 3) & 7) >= ((`t`.`fl_item` >> 3) & 7)) and (((`i`.`qa_item` >> 6) & 7) >= ((`t`.`fl_item` >> 6) & 7)) and (((`i`.`qa_item` >> 9) & 7) >= ((`t`.`fl_item` >> 9) & 7)) and (((`i`.`qa_item` >> 12) & 7) >= ((`t`.`fl_item` >> 12) & 7)) and (((`i`.`qa_item` >> 15) & 7) >= ((`t`.`fl_item` >> 15) & 7)) and (((`i`.`fl_slot` >> 0) & 3) <= ((`t`.`qa_slot` >> 0) & 3)) and (((`i`.`fl_slot` >> 2) & 3) <= ((`t`.`qa_slot` >> 2) & 3)) and (((`i`.`fl_slot` >> 4) & 3) <= ((`t`.`qa_slot` >> 4) & 3)) and (((`i`.`fl_slot` >> 6) & 3) <= ((`t`.`qa_slot` >> 6) & 3)) and (((`i`.`fl_slot` >> 8) & 3) <= ((`t`.`qa_slot` >> 8) & 3)) and (((`i`.`fl_slot` >> 10) & 3) <= ((`t`.`qa_slot` >> 10) & 3)) and (((`i`.`fl_slot` >> 12) & 3) <= ((`t`.`qa_slot` >> 12) & 3)) and (((`i`.`fl_slot` >> 14) & 3) <= ((`t`.`qa_slot` >> 14) & 3)) and (((`i`.`fl_slot` >> 16) & 3) <= ((`t`.`qa_slot` >> 16) & 3)) and (((`i`.`fl_slot` >> 18) & 3) <= ((`t`.`qa_slot` >> 18) & 3)) and (((`i`.`fl_slot` >> 20) & 3) <= ((`t`.`qa_slot` >> 20) & 3)) and ((`i`.`endx` >= now()) or (`i`.`endx` is null)) and (((`a`.`access_order` = 'White') and (`ac`.`entity_id` is not null)) or ((`a`.`access_order` = 'Black') and (`ac`.`entity_id` is null))) and (((`i`.`channel_order` = 'Black') and ((`ha`.`entity_id` is null) or (`hb`.`entity_id` is null))) or ((`i`.`channel_order` = 'White') and (`ha`.`entity_id` is not null) and (`hb`.`entity_id` is not null))) and (((`t`.`channel_order` = 'Black') and ((`ca`.`entity_id` is null) or (`cb`.`entity_id` is null))) or ((`t`.`channel_order` = 'White') and (`ca`.`entity_id` is not null) and (`cb`.`entity_id` is not null))) and (((`p`.`access_order` = 'White') and (`bc`.`entity_id` is not null)) or ((`p`.`access_order` = 'Black') and (`bc`.`entity_id` is null)))) */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `view_payment`
--

/*!50001 DROP VIEW IF EXISTS `view_payment`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb3 */;
/*!50001 SET character_set_results     = utf8mb3 */;
/*!50001 SET collation_connection      = utf8mb3_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50001 VIEW `view_payment` AS select `p`.`payment_id` AS `payment_id`,`p`.`status` AS `status`,`p`.`paytype_id` AS `entitytype_id`,`p`.`payment_id` AS `entity_id`,`a`.`adv_id` AS `adv_id`,`p`.`amount` AS `amount`,`p`.`ip` AS `ip`,`p`.`created` AS `created`,`d`.`paytype_value` AS `paytype_value`,concat(`a`.`firstname`,' ',`a`.`lastname`) AS `sender_name`,`a`.`adv_id` AS `sender_id`,unix_timestamp(`p`.`created`) AS `transaction_id` from ((`pay_payment` `p` join `def_paytype` `d` on((`p`.`paytype_id` = `d`.`paytype_id`))) join `adv` `a` on((`p`.`adv_id` = `a`.`adv_id`))) where (`p`.`paytype_id` in (1,2)) union select `p`.`payment_id` AS `payment_id`,`p`.`status` AS `status`,`p`.`paytype_id` AS `entitytype_id`,`cc`.`cc_id` AS `entity_id`,`p`.`adv_id` AS `adv_id`,`p`.`amount` AS `amount`,`p`.`ip` AS `ip`,`p`.`created` AS `created`,`d`.`paytype_value` AS `paytype_value`,`a`.`contact` AS `sender_name`,`a`.`address_id` AS `sender_id`,`cc`.`transaction_id` AS `transaction_id` from (((`pay_payment` `p` join `def_paytype` `d` on((`p`.`paytype_id` = `d`.`paytype_id`))) join `pay_cc` `cc` on(((`p`.`paytype_id` = 3) and (`p`.`payment_id` = `cc`.`payment_id`)))) join `add_address` `a` on((`cc`.`address_id` = `a`.`address_id`))) union select `p`.`payment_id` AS `payment_id`,`p`.`status` AS `status`,`p`.`paytype_id` AS `entitytype_id`,`cq`.`cheque_id` AS `entity_id`,`p`.`adv_id` AS `adv_id`,`p`.`amount` AS `amount`,`p`.`ip` AS `ip`,`p`.`created` AS `created`,`d`.`paytype_value` AS `paytype_value`,`a`.`contact` AS `sender_name`,`a`.`address_id` AS `sender_id`,`cq`.`transaction_id` AS `transaction_id` from (((`pay_payment` `p` join `def_paytype` `d` on((`p`.`paytype_id` = `d`.`paytype_id`))) join `pay_cheque` `cq` on(((`p`.`paytype_id` = 4) and (`p`.`payment_id` = `cq`.`payment_id`)))) join `add_address` `a` on((`cq`.`address_id` = `a`.`address_id`))) union select `p`.`payment_id` AS `payment_id`,`p`.`status` AS `status`,`p`.`paytype_id` AS `entitytype_id`,`w`.`wechat_id` AS `entity_id`,`p`.`adv_id` AS `adv_id`,`p`.`amount` AS `amount`,`p`.`ip` AS `ip`,`p`.`created` AS `created`,`d`.`paytype_value` AS `paytype_value`,`w`.`sender_name` AS `sender_name`,`w`.`sender_id` AS `sender_id`,`w`.`transaction_id` AS `transaction_id` from ((`pay_payment` `p` join `def_paytype` `d` on((`p`.`paytype_id` = `d`.`paytype_id`))) join `pay_wechat` `w` on(((`p`.`paytype_id` = 5) and (`p`.`payment_id` = `w`.`payment_id`)))) union select `p`.`payment_id` AS `payment_id`,`p`.`status` AS `status`,`p`.`paytype_id` AS `entitytype_id`,`a`.`alipay_id` AS `entity_id`,`p`.`adv_id` AS `adv_id`,`p`.`amount` AS `amount`,`p`.`ip` AS `ip`,`p`.`created` AS `created`,`d`.`paytype_value` AS `paytype_value`,`a`.`sender_name` AS `sender_name`,`a`.`sender_id` AS `sender_id`,`a`.`transaction_id` AS `transaction_id` from ((`pay_payment` `p` join `def_paytype` `d` on((`p`.`paytype_id` = `d`.`paytype_id`))) join `pay_alipay` `a` on(((`p`.`paytype_id` = 6) and (`p`.`payment_id` = `a`.`payment_id`)))) */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-01-13  2:14:29
