-- phpMyAdmin SQL Dump
-- version 5.0.2
-- https://www.phpmyadmin.net/
--
-- Host: v000306.adm.ds.fhnw.ch
-- Erstellungszeit: 23. Feb 2023 um 15:55
-- Server-Version: 10.3.31-MariaDB-1:10.3.31+maria~bionic-log
-- PHP-Version: 7.2.34-8+ubuntu16.04.1+deb.sury.org+1

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Datenbank: `tbbs`
--

-- --------------------------------------------------------

--
-- Tabellenstruktur für Tabelle `bagit`
--

CREATE TABLE `bagit` (
  `bagitid` bigint(20) NOT NULL,
  `name` varchar(255) COLLATE utf8_bin NOT NULL,
  `filesize` bigint(11) NOT NULL,
  `ingestfolder` varchar(512) COLLATE utf8_bin NOT NULL DEFAULT '',
  `sha512` char(128) COLLATE utf8_bin NOT NULL,
  `sha512_aes` char(128) COLLATE utf8_bin DEFAULT NULL,
  `start` timestamp NULL DEFAULT NULL,
  `end` timestamp NULL DEFAULT NULL,
  `baginfo` text COLLATE utf8_bin NOT NULL DEFAULT '',
  `report` text COLLATE utf8_bin DEFAULT NULL,
  `creator` varchar(256) COLLATE utf8_bin NOT NULL,
  `creationdate` timestamp NOT NULL DEFAULT current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Tabellenstruktur für Tabelle `bagit_location`
--

CREATE TABLE `bagit_location` (
  `bagitid` bigint(20) NOT NULL,
  `locationid` bigint(20) NOT NULL,
  `transfer_start` timestamp NULL DEFAULT NULL,
  `transfer_end` timestamp NULL DEFAULT NULL,
  `status` enum('ok','error') COLLATE utf8_bin NOT NULL,
  `message` text COLLATE utf8_bin DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Tabellenstruktur für Tabelle `bagit_test_location`
--

CREATE TABLE `bagit_test_location` (
  `bagit_location_testid` bigint(20) NOT NULL,
  `bagitid` bigint(20) NOT NULL,
  `testid` bigint(20) NOT NULL,
  `locationid` bigint(20) NOT NULL,
  `start` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `end` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `status` set('passed','failed') COLLATE utf8_bin NOT NULL,
  `data` longtext COLLATE utf8_bin DEFAULT NULL,
  `message` text COLLATE utf8_bin DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Tabellenstruktur für Tabelle `content`
--

CREATE TABLE `content` (
  `contentid` bigint(20) NOT NULL,
  `bagitid` bigint(20) NOT NULL,
  `zippath` varchar(2048) COLLATE utf8_bin DEFAULT NULL,
  `diskpath` varchar(2048) COLLATE utf8_bin NOT NULL,
  `filesize` bigint(11) NOT NULL,
  `checksums` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `sha256` char(64) COLLATE utf8_bin DEFAULT NULL,
  `sha512` char(128) COLLATE utf8_bin DEFAULT NULL,
  `md5` char(32) COLLATE utf8_bin DEFAULT NULL,
  `mimetype` varchar(255) COLLATE utf8_bin DEFAULT NULL,
  `width` int(11) DEFAULT NULL,
  `height` int(11) DEFAULT NULL,
  `duration` int(11) DEFAULT NULL,
  `indexer` longtext COLLATE utf8_bin DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Tabellenstruktur für Tabelle `location`
--

CREATE TABLE `location` (
  `locationid` bigint(20) NOT NULL,
  `name` varchar(255) COLLATE utf8_bin NOT NULL,
  `path` varchar(1024) COLLATE utf8_bin NOT NULL DEFAULT '',
  `testinterval` varchar(64) COLLATE utf8_bin NOT NULL,
  `params` text COLLATE utf8_bin NOT NULL DEFAULT '',
  `encrypted` tinyint(4) NOT NULL DEFAULT 0,
  `quality` float NOT NULL DEFAULT 0,
  `costs` float NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Tabellenstruktur für Tabelle `test`
--

CREATE TABLE `test` (
  `testid` bigint(20) NOT NULL,
  `name` varchar(255) COLLATE utf8_bin NOT NULL,
  `description` text COLLATE utf8_bin NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Tabellenstruktur für Tabelle `test_locationtype`
--

CREATE TABLE `test_locationtype` (
  `testid` bigint(20) NOT NULL,
  `locationtype` enum('init','local','ssh') COLLATE utf8_bin NOT NULL,
  `enctype` enum('both','plain','encrypted') COLLATE utf8_bin NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

--
-- Indizes der exportierten Tabellen
--

--
-- Indizes für die Tabelle `bagit`
--
ALTER TABLE `bagit`
  ADD PRIMARY KEY (`bagitid`),
  ADD UNIQUE KEY `name` (`name`) USING BTREE,
  ADD KEY `creationdate` (`creationdate`);

--
-- Indizes für die Tabelle `bagit_location`
--
ALTER TABLE `bagit_location`
  ADD PRIMARY KEY (`bagitid`,`locationid`),
  ADD KEY `locationid` (`locationid`);

--
-- Indizes für die Tabelle `bagit_test_location`
--
ALTER TABLE `bagit_test_location`
  ADD PRIMARY KEY (`bagit_location_testid`),
  ADD KEY `testid` (`testid`),
  ADD KEY `locationid` (`locationid`),
  ADD KEY `start` (`start`),
  ADD KEY `end` (`end`),
  ADD KEY `status` (`status`),
  ADD KEY `bagitid` (`bagitid`);

--
-- Indizes für die Tabelle `content`
--
ALTER TABLE `content`
  ADD PRIMARY KEY (`contentid`),
  ADD KEY `bagitid` (`bagitid`);

--
-- Indizes für die Tabelle `location`
--
ALTER TABLE `location`
  ADD PRIMARY KEY (`locationid`) USING BTREE,
  ADD UNIQUE KEY `name` (`name`) USING BTREE,
  ADD KEY `encrypted` (`encrypted`);

--
-- Indizes für die Tabelle `test`
--
ALTER TABLE `test`
  ADD PRIMARY KEY (`testid`),
  ADD UNIQUE KEY `name` (`name`);

--
-- Indizes für die Tabelle `test_locationtype`
--
ALTER TABLE `test_locationtype`
  ADD PRIMARY KEY (`testid`,`locationtype`);

--
-- AUTO_INCREMENT für exportierte Tabellen
--

--
-- AUTO_INCREMENT für Tabelle `bagit`
--
ALTER TABLE `bagit`
  MODIFY `bagitid` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT für Tabelle `bagit_test_location`
--
ALTER TABLE `bagit_test_location`
  MODIFY `bagit_location_testid` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT für Tabelle `content`
--
ALTER TABLE `content`
  MODIFY `contentid` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT für Tabelle `location`
--
ALTER TABLE `location`
  MODIFY `locationid` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT für Tabelle `test`
--
ALTER TABLE `test`
  MODIFY `testid` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- Constraints der exportierten Tabellen
--

--
-- Constraints der Tabelle `bagit_location`
--
ALTER TABLE `bagit_location`
  ADD CONSTRAINT `bagit_location_ibfk_1` FOREIGN KEY (`bagitid`) REFERENCES `bagit` (`bagitid`) ON DELETE CASCADE,
  ADD CONSTRAINT `bagit_location_ibfk_2` FOREIGN KEY (`locationid`) REFERENCES `location` (`locationid`) ON DELETE CASCADE;

--
-- Constraints der Tabelle `bagit_test_location`
--
ALTER TABLE `bagit_test_location`
  ADD CONSTRAINT `bagit_test_location_ibfk_1` FOREIGN KEY (`bagitid`) REFERENCES `bagit` (`bagitid`) ON DELETE CASCADE,
  ADD CONSTRAINT `bagit_test_location_ibfk_2` FOREIGN KEY (`locationid`) REFERENCES `location` (`locationid`) ON DELETE CASCADE;

--
-- Constraints der Tabelle `content`
--
ALTER TABLE `content`
  ADD CONSTRAINT `content_ibfk_1` FOREIGN KEY (`bagitid`) REFERENCES `bagit` (`bagitid`) ON DELETE CASCADE;

--
-- Constraints der Tabelle `test_locationtype`
--
ALTER TABLE `test_locationtype`
  ADD CONSTRAINT `test_locationtype_ibfk_1` FOREIGN KEY (`testid`) REFERENCES `test` (`testid`) ON DELETE CASCADE;
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
