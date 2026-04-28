-- Reversa de 006_people.up.sql.
-- Drop en orden inverso: primero asignaciones (FK a vehicles), luego vehiculos.

DROP TABLE IF EXISTS unit_vehicle_assignments;
DROP TABLE IF EXISTS vehicles;
