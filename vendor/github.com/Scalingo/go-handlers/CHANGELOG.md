# Changelog

## v1.4.2

* fix(error_middleware): better detection of JSON Content-Type
* fix(error_middleware): do not write a body if it has already been written
* Bump github.com/gofrs/uuid from 4.1.0+incompatible to 4.2.0+incompatible

## v1.4.1

* Bump github.com/sirupsen/logrus from 1.7.0 to 1.8.1
* fix(error_middleware): ValidationError shouldn't be logged as Error
* Bump github.com/Scalingo/go-utils/logger from 1.0.0 to 1.1.0
* chore(deps): upgrade github.com/gofrs/uuid from 3.4.0 to 4.1.0

## v1.4.0

* Don't return an invalid auth error for AuthMiddleware and add the ability to encode the error in json instead of text
  [#27](https://github.com/Scalingo/go-handlers/pull/27)

## v1.3.1

* Bump github.com/stretchr/testify from 1.5.1 to 1.7.0 #24
* Bump github.com/gorilla/mux from 1.7.4 to 1.8.0 #25
* Bump github.com/gofrs/uuid from 3.2.0+incompatible to 3.4.0+incompatible #26

## v1.3.0 - 14 Apr 2019

* Update all dependencies
* Use github.com/gofrs/uuid instead of unsecure github.com/satori/go.uuid

## v1.2.6 - 5 Jul 2019

* Change UUID generation lib

## v1.2.5 - 12 Apr 2019

* Error middleware handle the case of ValidationErrors

## v1.2.4 - 9 Jan 2018

* Accept github.com/Scalingo/go-utils.ErrCtx to pass ctx to error middleware

## v1.2.3 - 29 Nov 2017

* Add ability to create logging level filters in the LoggerMiddleware
* Possibility to create a router without any logging middleware

## v1.2.1 - v1.2.2

* Accept a logrus.FieldLogger instead of a *logrus.Logger when creating a router

## v1.2.0

* First tagged version
