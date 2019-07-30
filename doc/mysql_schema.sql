DROP TABLE IF EXISTS `datasets`;
CREATE TABLE `datasets` (
  `dataset_id` int(11) NOT NULL AUTO_INCREMENT,
  `experiment_id` int(11) NOT NULL,
  `processing_id` int(11) NOT NULL,
  `tier_id` int(11) NOT NULL,
  `tstamp` int(11) NOT NULL,
  PRIMARY KEY (`dataset_id`),
  UNIQUE KEY `tstamp` (`tstamp`),
  KEY `fk_eid` (`experiment_id`),
  KEY `fk_pid` (`processing_id`),
  KEY `fk_tid` (`tier_id`),
  CONSTRAINT `fk_eid` FOREIGN KEY (`experiment_id`) REFERENCES `experiments` (`experiment_id`) ON UPDATE CASCADE,
  CONSTRAINT `fk_pid` FOREIGN KEY (`processing_id`) REFERENCES `processing` (`processing_id`) ON UPDATE CASCADE,
  CONSTRAINT `fk_tid` FOREIGN KEY (`tier_id`) REFERENCES `tiers` (`tier_id`) ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

--
-- Table structure for table `experiments`
--

DROP TABLE IF EXISTS `experiments`;
CREATE TABLE `experiments` (
  `experiment_id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  PRIMARY KEY (`experiment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

--
-- Table structure for table `files`
--

DROP TABLE IF EXISTS `files`;
CREATE TABLE `files` (
  `file_id` int(11) NOT NULL AUTO_INCREMENT,
  `dataset_id` int(11) NOT NULL,
  `name` varchar(100) NOT NULL,
  PRIMARY KEY (`file_id`),
  KEY `fk_did` (`dataset_id`),
  CONSTRAINT `fk_did` FOREIGN KEY (`dataset_id`) REFERENCES `datasets` (`dataset_id`) ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

--
-- Table structure for table `processing`
--

DROP TABLE IF EXISTS `processing`;
CREATE TABLE `processing` (
  `processing_id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  PRIMARY KEY (`processing_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

--
-- Table structure for table `tiers`
--

DROP TABLE IF EXISTS `tiers`;
CREATE TABLE `tiers` (
  `tier_id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  PRIMARY KEY (`tier_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
