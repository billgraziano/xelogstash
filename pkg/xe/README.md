Extended Event Package Notes
============================


Functions
---------
* Parse() -> unmarshals an XML string
* getDataValue -> getValue
* getActionValue -> getValue
* getValue - accepts field name, data type, and string value and returns a GO type


Types
-----
* plan_handle: binary_data
* query_hash: uint64
* query_hash_signed: int64

Hashes and Plans
----------------
```xml
  <action name="plan_handle" package="sqlserver">
    <value>06000100878dd010b0f12822c501000001000000000000000000000000000000000000000000000000000000</value>
  </action>
  <action name="query_hash" package="sqlserver">
    <value>3279475884177764727</value>
  </action>
  <action name="query_hash_signed" package="sqlserver">
    <value>3279475884177764727</value>
  </action>
  <action name="query_plan_hash" package="sqlserver">
    <value>6189487012607264618</value>
  </action>
  <action name="query_plan_hash_signed" package="sqlserver">
    <value>6189487012607264618</value>
  </action>
  ```
