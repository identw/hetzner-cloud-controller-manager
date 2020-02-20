Changelog
=========

v0.0.4
------
 * Fix problem with requests rate limit for Hrobot API (200 requests per hour)
 * Servers from hrobot api are now cached in memory and updated with the period `HROBOT_PERIOD` seconds

v0.0.3 
------
 * add capability: exclude the removal of nodes that belong to other providers

v0.0.2
------
* Fix bug: invalid memory address or nil pointer dereference if server not found in hrobot

v0.0.1
------
* Initial
