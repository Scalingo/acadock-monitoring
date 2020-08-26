# Changelog

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
