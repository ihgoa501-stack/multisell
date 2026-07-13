DROP TRIGGER IF EXISTS sourcing_watch_alert_no_delete ON sourcing_1688_watch_alert;
DROP TRIGGER IF EXISTS sourcing_watch_alert_no_update ON sourcing_1688_watch_alert;
DROP FUNCTION IF EXISTS prevent_sourcing_watch_alert_mutation();
DROP TABLE IF EXISTS sourcing_1688_watch_alert;
DROP TABLE IF EXISTS sourcing_1688_watch_refresh_run;
DROP TABLE IF EXISTS sourcing_1688_watch_subscription;
