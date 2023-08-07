Feature: Documentation
  In order to test the documentation feature
  I should create a blueprint
  and the containers should be running

  Scenario: Documentation
    Given I have a running blueprint
    Then the following resources should be running
      | name                    |
      | resource.docs.docs      |
    And a HTTP call to "http://docs.docs.shipyard.run/" should result in status 200

  #Scenario: Documentation with different version
  #  Given I have a running blueprint using version "v0.2.16"
  #  Then the following resources should be running
  #    | name                    | type      |
  #    | docs                    | docs      |
  #  And a HTTP call to "http://docs.docs.shipyard.run:8080/" should result in status 200