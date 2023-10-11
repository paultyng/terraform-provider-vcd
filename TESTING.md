# Testing terraform-provider-vcd

## Table of contents

- [Meeting prerequisites: Building the test environment](#meeting-prerequisites-building-the-test-environment)
- [Running tests](#running-tests)
- [Tests split by feature set](#tests-split-by-feature-set)
- [Adding new tests](#adding-new-tests)
  - [Parallelism considerations](#parallelism-considerations)
- [Binary testing](#binary-testing)
- [Handling failures in binary tests](#handling-failures-in-binary-tests)
- [Upgrade testing](#upgrade-testing)
- [Custom terraform scripts](#custom-terraform-scripts)
- [Conditional running of tests](#conditional-running-of-tests)
- [Tests with multiple providers](#tests-with-multiple-providers)
- [Partitioned tests](#partitioned-tests)
- [Leftovers removal](#leftovers-removal)
- [Environment variables and corresponding flags](#environment-variables-and-corresponding-flags)
- [Troubleshooting code issues](#troubleshooting-code-issues)

## Meeting prerequisites: Building the test environment

To run the tests, your vCD needs to have the following:

* (1) An external network, with at least one, but preferably two allocatable IP addresses;
* (2) An organization
    * (3) An organization administrator user
    * (4) A Virtual Data Center (VDC) with enough resources to build several vApps and VMs, and preferably 2 storage profiles
        * (5) an Edge gateway with a default gateway
    * (6) A catalog
    * (7) A vApp template in the catalog

Optionally, to run tests in [go-vcloud-director](https://github.com/vmware/go-vcloud-director), you will also need one or
two Org VDC networks and a media item.


This is how you can build the environment using Terraform itself.

The above entities are defined in the test configuration file (see `sample_vcd_test_config.json` for the full list of
entities to provide). Here is an excerpt of where these names come from:

```
{
  "vcd": {
     "org": "datacloud",                    // (2)
     "vdc": "datacloudvdc",                 // (4)
     "catalog": {
       "name": "datacloudcat",              // (6)
       "catalogItem": "photon-hw11"         // (7)
     },
     [...]
   },
   "networking": {
     "edgeGateway": "gwdatacloud",           // (5)
     "externalNetwork": "extnetdatacloud",   // (1)
     [...]
     }
   },
   [...]
 }

```

The vCD environment builder uses the test system information to create these entities. When triggered, it will take the
names of the entities and some of the configuration parameters from the regular test configuration file. It will then
integrate such data with additional information that is only used for the environment build, and it's listed under the
`testEnvBuild` section in the file:

```
{
   [...]
  "testEnvBuild": {
    // Indispensable data, which is not normally used in the tests but is mandatory to create the
    // environment

    "gateway": "GATEWAY_IP",
    "netmask": "255.255.224.0",
    "externalNetworkStartIp": "STARTING_IP",
    "externalNetworkEndIp": "END_IP",
    "dns1": "DNS_IP",

    // Optional data. If not provided, the corresponding values
    // from the rest of the configuration file will be used

    "dns2": "8.8.4.4",
    "externalNetworkPortGroup": "VM Network",
    "externalNetworkPortGroupType": "NETWORK",
    "ovaPath": "/path/to/photon-hw11-3.0-26156e2.ova",
    "media_path":  "/path/to/test.iso",

    // extra data. If not provided, these resources will not be created
    // These items can be used for go-vcloud-director testing

    "mediaName": "test_media",
    "storageProfile2": "Development",
    "routedNetwork": "net_datacloud_r",
    "isolatedNetwork": "net_datacloud_i",
    "directNetwork": "net_datacloud_d"
    "orgUser": "company-admin",
    "orgUserPassword": "secretpwd"
  }
}
```

When this section is filled, the test system has all the information to create all the elements. When you run the
command `make test-binary-prepare`, among the files ready to run there will be one named
`cust.full-env.tf`, containing all the information to populate your vCD with the test resources.
**IMPORTANT**: this procedure assumes that **the regular part of the configuration file is filled with all the correct
information**. The list of fields given as mandatory above is not sufficient to create a working environment, if
information such as provider VDC or vCenter are not provided.

There are also a couple of commands that prepare and execute the terraform script for you:

```bash
$ make test-env-init && make test-env-apply
```

Unlike other commands executed during `make test-binary`, these ones only run the `init` and `apply` stage of the script processing,
producing a ready-to-use environment in a few minutes.
If the commands were successful, you are ready to run the acceptance tests:

```bash
$ make testacc
```

After it runs, this test will leave the vCD in its initial state, i.e. with the resources created by `test-env-build`.
If you want to clean up the vCD, you can run:

```bash
$ make test-env-destroy
```
which will wipe out everything that was created with `make test-env-apply`.

The build-up tasks run inside a dedicated directory below `./vcd/test-artifacts`. The directory name is made up of the
string `test-ebv-build` + the name or IP of the vCD. For example, for vcd *10.178.7.96*, the directory will be
`test-env-build-10-178-7-96`. For a vCD named *example-vcd.my_company.com*, the directory will be
`test-env-build-example-vcd-my_company-com`. This naming convention prevents overwriting the build-up files and allows
users to retrieve the build-up configuration files and run operations manually on them, even after different vCD were
configured.


## Running tests

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```

The acceptance tests will run against your own vCloud Director setup, using the configuration in your file `./vcd/vcd_test_config.json`
See the file `./vcd/sample_vcd_test_config.json` for an example of which variables need to be defined.

Each test in the suite will write a Terraform configuration file inside `./vcd/test-artifacts`, named after the
tests. For example: `vcd.TestAccVcdNetworkDirect.tf`

The test suite will try to minimize the amount of resources to create. If no catalog and vApp
template (`catalogItem`) are defined in the configuration file, new ones will be created and removed at the end of
the test. You can choose to preserve catalog and vApp template across runs (use the `preserve` field in the
configuration file).


Both the (short) test and the acceptance test include a run of the `unit` tests, i.e. tests that check the correctness
of the code without need for a live vCD.

You can run the unit tests directly with

```sh
make testunit
```

## Tests split by feature set

The tests can run with several tags that define which components are tested.
Using the Makefile, you can run one of the following:

```bash
make testcatalog
make testnetwork
make testextnetwork
make testgateway
make testvapp
make testvm
```

For more options, you can run manually in `./vcd`
When running `go test` without tags, you'll get a list of tags that are available.

```
$ go test -v .
=== RUN   TestTags
 --- FAIL: TestTags (0.00s)
     api_test.go:87: # No tags were defined
     api_test.go:62:
         # -----------------------------------------------------
         # Tags are required to run the tests
         # -----------------------------------------------------

         At least one of the following tags should be defined:

            * ALL :       Runs all the tests
            * functional: Runs all the acceptance tests
            * unit:       Runs unit tests that don't need a live vCD

            * catalog:    Runs catalog related tests (also catalog_item, media)
            * disk:       Runs disk related tests
            * network:    Runs network related tests
            * gateway:    Runs edge gateway related tests
            * org:        Runs org related tests
            * user:       Runs user related tests
            * vapp:       Runs vapp related tests
            * vdc:        Runs vdc related tests
            * vm:         Runs vm related tests

         Examples:

           go test -tags unit -v -timeout=45m .
           go test -tags functional -v -timeout=45m .
           go test -tags catalog -v -timeout=15m .
           go test -tags "org vdc" -v -timeout=5m .

         Tagged tests can also run using make
           make testunit
           make testacc
           make testcatalog
 FAIL
 FAIL	github.com/vmware/terraform-provider-vcd/v2/vcd	0.017s
```

## Adding new tests

All tests need to have a build tag. The tag should be the first line of the file, followed by a blank line

```go
// +build functional featurename ALL

package vcd
```

Tests that integrate in the functional suite use the tag `functional`. Using that tag, we can run all functional tests
at once.
We define as `functional` the tests that need a live vCD to run.

1. The test should always define the `ALL` tag:

* ALL :       Runs all the tests

2. The test should also always define either the `unit` or `functional` tag:

* functional: Runs all the tests that use a live vCD (acceptance tests)
* unit:       Runs unit tests that do not need a live vCD

3. Finally, the test should always define the feature tag. For example:

* catalog:    Runs catalog related tests (also `catalog_item`, `media`)
* vapp:       Runs vapp related tests

The `ALL` tag includes tests that use a different framework. At the moment, this is useful to run a global compilation test.
Depending on which additional tests we will implement, we may change the dependency on the `ALL` tag if we detect
clashes between frameworks.

If the test file defines a new feature tag (i.e. one that has not been used before) the file should also implement an
`init` function that sets the tag in the global tag list.
This information is used by the main tag test in `api_test.go` to determine which tags were activated.

```go
func init() {
	testingTags["newtag"] = "filename_test.go"
}
```

**VERY IMPORTANT**: if we add a test that runs using a different tag (i.e. it is not included in `functional` tests), we need
to add such test to GNUMakefile under `make test` and `make testacc`. **The general principle is that `make test` and `make testacc` run all tests**. If this can't be
achieved by adding the new test to the `functional` tag (perhaps because we foresee framework conflicts), we need to add the
new test as a separate command.
For example, the unit test run as a command before the acceptance test:

```
testacc: testunit
	@sh -c "'$(CURDIR)/scripts/runtest.sh' acceptance"
```

### Parallelism considerations

When writing Terraform acceptance tests there are two ways to define tests. Either using
`resource.Test` or `resource.ParallelTest`. The former runs tests sequentially one by one while the
later runs all the tests (which are defined to be run in parallel) instantiated this way in
parallel. This is useful because it can speed up total test execution time. However one must be sure
that the tests defined for parallel run are not clashing with each other.

By default `make testacc` runs acceptance tests with parallelism enabled (for the tests which are
defined with `resource.ParallelTest`). If there is a need to troubleshoot or simply force the tests
to run sequentially - `make seqtestacc` can be used to achieve it.

## Binary testing

By *binary testing* we mean the tests that run using Terraform binary executable, as opposed to running the test through the Go framework.
This test runs the same tasks that run in the acceptance test, but instead of running them directly, they are fed to the
terraform tool through a shell script, and for every test we run

* `terraform init`
* `terraform plan`
* `terraform apply -auto-approve`
* `terraform plan -detailed-exitcode` (for ensuring that `plan` is empty right after `apply`)
* `terraform destroy -auto-approve`

The test runs from GNUMakefile, using

```bash
make test-binary
```

All the tests run unattended, stopping only if there is an error.

It is possible to customise running of the binary tests by preparing them and then running the test script from the `tests-artifacts` directory:

```bash
make test-binary-prepare
[...]

cd ./vcd/test-artifacts
./test-binary.sh help

# OR
./test-binary.sh pause verbose

# OR
./test-binary.sh pause verbose tags "catalog gateway"
```

The "pause" option will stop the test after every call to the terraform tool, waiting for user input.

When the test runs unattended, it is possible to stop it gracefully by creating a file named `pause` inside the
`test-artifacts` directory. When such file exists, the test execution stops at the next `terraform` command, waiting
for user input.

## Handling failures in binary tests

When one test fails, the binary test script will attempt to recover it, by running `terraform destroy`. If the recovery
fails, the whole test halts. If recovery succeeds, the names of the failed test are recorded inside 
`./vcd/test-artifacts/failed_tests.txt` and the summary at the end of the test will show them.

If the test runs with `make test-binary`, the output is captured inside `./vcd/test-artifacts/test-binary-TIME.txt` (where
`TIME` has the format `YYYY-MM-DD-HH-MM`). To see the actual failure, open the output file and search for the name of
the test that failed.

For example, the test ends with this annotation :

```
# ---------------------------------------------------------
# Operations dir: /path/to/terraform-provider-vcd/vcd/test-artifacts/tmp
# Started:        Thu Mar 12 14:10:43 CET 2020
# Ended:          Thu Mar 12 14:12:16 CET 2020
# Elapsed:        1m:33s (93 sec)
# exit code:      0
# ---------------------------------------------------------
# ---------------------------------------------------------
# FAILED TESTS    4
# ---------------------------------------------------------
Thu Mar 12 14:11:02 CET 2020 - vcd.TestUser_test_user_catalog_author_basic.tf (apply)
Thu Mar 12 14:11:08 CET 2020 - vcd.TestUser_test_user_catalog_author_basic.tf (plancheck)
Thu Mar 12 14:11:30 CET 2020 - vcd.TestUser_test_user_admin_basic.tf (apply)
Thu Mar 12 14:11:36 CET 2020 - vcd.TestUser_test_user_admin_basic.tf (plancheck)
# ---------------------------------------------------------
```

In the output file (in the directory `./vcd/test-artifacts`), look for `vcd.TestUser_test_user_catalog_author_basic.tf`
and you will see the operations occurring with the actual errors.

## Upgrade testing

We can test that resources updated in the current release don't break the behavior of the same resources that were
created with a previous version of the provider.

The command `make test-upgrade` will do the following:

1. Fetch the tags from main
2. Check out the previous version (using the tag corresponding to the version stored in the file `PREVIOUS_VERSION`)
3. Run `make test-binary-prepare` using the previous version
4. Back to the current version, build the latest plugin
5. Run the binary tests, with the following behavior for each script:
   5a. Run `terraform init`, `terraform plan`, and `terraform apply` using the previous version plugin
   5b. Run `terraform plan -detailed-exitcode` and `terraform destroy` using the current version plugin

This test ensures that the resources created with the previous version don't have unexpected changes when a plan is
performed with the next version.

## Custom terraform scripts

The commands `make test-binary-prepare` and `make test-binary` have the added benefit of compiling custom Terraform scripts located in `./vcd/test-templates`.
These tests are similar to the ones produced by the testing framework, but unlike the standard ones, they can be edited by users. And users can also remove and add files to suit their purposes.

The files in `test-templates` are not executable directly by `terraform`: they need to be processed (which happens during `make test-binary-prepare`) and their placeholders expanded to the values taken from the configuration file.

A **placeholder** is a label enclosed in double braces and prefixed by a dot, such as `{{.LabelName}}`.
The template processor will replace that label with the corresponding value taken from the test configuration file.
See `sample_vcd_test_config.json` for a description of all fields.

This is the list of placeholders that you can use in template files:

Label                        | Field in `vcd_test_config.json`
:----------------------------|:------------------------------------------
Org                          | vcd.org
Vdc                          | vcd.vdc
ProviderVdc                  | vcd.providerVdc.name
NetworkPool                  | vcd.providerVdc.networkPool
StorageProfile               | vcd.providerVdc.storageProfile
Catalog                      | vcd.catalog.name
CatalogItem                  | vcd.catalog.catalogItem
OvaPath                      | ova.OvaPath
MediaPath                    | media.MediaPath
MediaUploadPieceSize         | media.UploadPieceSize
MediaUploadProgress          | media.UploadProgress
OvaDownloadUrl               | ova.ovaDownloadUrl
OvaTestFileName              | ova.ovaTestFileName
OvaUploadProgress            | ova.uploadProgress
OvaUploadPieceSize           | ova.uploadPieceSize
OvaPreserve                  | ova.preserve
LoggingEnabled               | logging.enabled
LoggingFileName              | logging.logFileName
EdgeGateway                  | networking.edgeGateway
SharedSecret                 | networking.sharedSecret
ExternalNetwork              | networking.externalNetwork
ExternalNetworkPortGroup     | networking.externalNetworkPortGroup
ExternalNetworkPortGroupType | networking.externalNetworkPortGroupType
ExternalIp                   | networking.externalIp
InternalIp                   | networking.internalIp
Vcenter                      | networking.vcenter
LocalIp                      | networking.local.localIp
LocalGateway                 | networking.local.localSubnetGateway
PeerIp                       | networking.peer.peerIp
PeerGateway                  | networking.peer.peerSubnetGateway
MaxRetryTimeout              | provider.maxRetryTimeout
AllowInsecure                | provider.allowInsecure
ProviderSysOrg               | provider.sysOrg
ProviderUrl                  | provider.url
ProviderUser                 | provider.user
ProviderPassword             | provider.password
ProviderSamlUser             | provider.samlUser
ProviderSamlPassword         | provider.samlPassword
ProviderSamlRptId            | provider.samlCustomRptId


The files generated from `./vcd/test-templates` will end up in `./vcd/test-artifacts`, and you will recognize them because their name will start by `cust.` instead of `vcd.`, and they all use the tag `custom`.

Note that the template files should **not** have a `provider` section, as it is created by the template processor.
Inside the template, you can indicate the need for specific `terraform` options, by inserting one or more comments containing `init-options`, `plan-options`, `apply-options`, or `destroy-options`. The options, if indicated, will be added to the corresponding `terraform` command. For example:

```
# apply-options -no-color
# destroy-options -no-color
```
When running `terraform apply` and `terraform destroy`, the option `-no-color` will be added to the command line.

To run these tests, you go inside `test-artifacts` and execute:

```bash
./test-binary.sh names "cust*.tf" [options]

# or

./test-binary.sh names "*.tf" tags custom [options]

# or

./test-binary.sh names cust.specific-file-name.tf [options]
```

The execution then proceeds as explained in [Binary testing](#Binary-testing).

## Conditional running of tests

The whole test suite takes several hours to run. If some errors happen during the run, we need to clean up and try again
from the beginning, which is not always convenient.
There are a few tags that help us gain some control on the flow:

* `-vcd-pre-post-checks`    Global switch enabling checks before and after tests (false). Also activated by using any of the flags below.
* `-vcd-re-run-failed`      Run only tests that failed in a previous run (false)
* `-vcd-remove-test-list`   Remove list of test runs (false)
* `-vcd-show-count`         Show number of pass/fail tests (false)
* `-vcd-show-elapsed-time`  Show elapsed time since the start of the suite in pre and post checks (false)
* `-vcd-show-timestamp`     Show timestamp in pre and post checks (false)
* `-vcd-skip-pattern`       Skip tests that match the pattern (implies vcd-pre-post-checks ()

When `-vcd-pre-post-checks` is used, we have several advantages:

1. After each successful test, the test name gets recorded in a file `vcd_test_pass_list_{VCD_IP}.txt`, and each failed
   test goes to `vcd_test_fail_list_{VCD_IP}.txt`. When running the suite on the same VCD a second time, all tests in
   the `pass` list are skipped. If the test run was interrupted (see #2 below), we can only run the tests that did not
   run in the previous attempt.
2. We can **gracefully** interrupt the tests by creating a file `skip_vcd_tests` in the `./vcd` directory. 
   When this file is found by the pre-run routine, all the tests are skipped. The file `skip_vcd_tests` will be removed
   automatically at the next run.
3. We can skip one or more tests conditionally, using `-vcd-skip-pattern="{REGEXP}"`. All the test with a name that
   matches the pattern are skipped.
4. We can re-run only the tests that failed in the previous run, using `-vcd-re-run-failed`.
5. We can add monitoring information with `-vcd-show-count`, `-vcd-show-elapsed-time`, `-vcd-show-timestamp`.

If we use `-vcd-pre-post-checks` and the run was successful, the next run will skip all tests, because the test names
would be all found in `vcd_test_pass_list_{VCD_IP}.txt`. To run again the test from scratch, we could either remove
the file manually, or use the tag `-vcd-remove-test-list`.

**VERY IMPORTANT**: for the conditional running to work, each test must have a call to `preTestChecks(t)`  at the beginning
and to `postTestChecks(t)` right before the end.

## Tests with multiple providers

When the test requires multiple providers (such as system administrator + tenant or two different tenants with an
optional system administrator), we can take advantage of the capability of setting multiple providers as `ProviderFactories`,
using a pre-defined function and several conventions:

1. Set the test as being only runnable by system administrator (`skipIfNotSysAdmin`). The Org user roles will be defined
   by the provider names (see item #3).
2. Add the provider factories (`ProviderFactories: buildMultipleProviders(),`)
3. Assign an explicit provider to every resource or data source, using `provider = vcd` for system administrator, 
   and `provider = vcdorg1` and `provider = vcdorg2` for tenants using the first and second Org in your VCD (e.g. "testorg" and "testorg-1").
4. The provider names must not be changed. Also, do not use the expressions `vcdorg1` or `vcdorg2` in any other
   test that don't require multiple providers.

The test framework does not support aliases. Therefore, the test that runs in the integrated environment will be slightly
different from the text that gets written to the files in `test-artifacts`, where the absolute provider names get
converted to aliases, and an explicit provider definition for the Org users is added to the script.
Look at `TestResourceInfoProviders` to see a full example of how to use the method described in this section.

**CAVEAT**: when using `buildMultipleProviders()`, you must make sure that the system provider (`vcd`) is used at least
once in the HCL script. If it is not, the variable `testAccProvider` may not get initialised, and if that happens,
test checks that use the expression `conn := testAccProvider.Meta().(*VCDClient)` will panic.

## Partitioned tests

We can run tests in a partitioned way, by splitting the tests across several VCDs (nodes) each of which will run a given portion of the tests.

To activate this modality, we use the following options:

* `-vcd-partitions=N` indicates the number of partitions running the test suite.
* `-vcd-partition-node=N` indicates which node the current VCD will run.
* `-vcd-partition-tests-file=FileName` (optional) provides the list of tests to run

When partition mode is enabled, the current node will skip all the tests that are not assigned to it.

The test assignment, by default, happens by getting the list of all tests, sorting them alphabetically, and then assigning them to each node in sequence. 
If a tests file name was provided, the node will just use such file as its assignment.

Each node produces several files:

* `{BUILD_NUMBER}-tests-retrieved-in-node-{NODE_NUMBER}-{VCD_VERSION}.txt` All the tests found in the current node
* `{BUILD_NUMBER}-tests-planned-in-node-{NODE_NUMBER}-{VCD_VERSION}.txt` The tests that will run in the current node
* `{BUILD_NUMBER}-tests-processed-in-node-{NODE_NUMBER}-{VCD_VERSION}.txt` The tests that have been processed in this node
* `{BUILD_NUMBER}-out-{NODE_NUMBER}.txt` A sentinel file signifying that the tests in the current node are finished. It contains the error code of the run (0=success, 1=failure)

`{BUILD_NUMBER}` is the number of the build available as environment variable in Jenkins jobs. If we run the test outside
that environment, the test software will replace the build number with `LOCAL`. 
For compatibility with other tools, it is recommended to set a dummy `BUILD_NUMBER` when running tests locally.

NOTE: when running partitioned tests with value for `-tags` other than `functional` or `ALL`, there will be a discrepancy 
between the tests collected in `{BUILD_NUMBER}-tests-planned-in-node-{NODE_NUMBER}-{VCD_VERSION}.txt` and the ones that
will be processed. The planned tests are collected without any `tags` consideration. Thus, the count of processed tests
will likely be much lower than the number of tests "planned".

## Leftovers removal

After the test stuite runs, an automated process will scan the VCD and remove any resources that may have been
left behind because od test failure or environment issues.
The procedure can be skipped by using the flag `-vcd-skip-leftovers-removal`. If you want the operation to omit
details of the scanning, you can use `-vcd-silent-leftovers-removal`.

To run the removal only, without running the full suite, use the command

```
$ go test -tags functional -run RemoveLeftovers # or the name of any non-existing test
```


## Environment variables and corresponding flags

There are several environment variables that can affect the tests. Many of them have a corresponding flag
that can be used in combination with the `go test` command. You can see them using the `-vcd-help` flag.

* `TF_ACC=1` enables the acceptance tests. It is also set when you run `make testacc`.
* `GOVCD_DEBUG=1` (`-vcd-debug`) enables debug output of the test suite
* `GOVCD_TRACE=1` (`-vcd-trace`) enables function calls tracing
* `VCD_SKIP_TEMPLATE_WRITING=1`  (`-vcd-skip-template-write`) skips the production of test templates into `./vcd/test-artifacts`
* `VCD_ADD_PROVIDER=1` (`-vcd-add-provider`) Adds the full provider definition to the snippets inside `./vcd/test-artifacts`.
   **WARNING**: the provider definition includes your vCloud Director credentials.
* `VCD_CONFIG=FileName` sets the file name for the test configuration file.
* `REMOVE_ORG_VDC_FROM_TEMPLATE` (`-vcd-remove-org-vdc-from-template`) is a quick way of enabling an alternate testing mode:
When `REMOVE_ORG_VDC_FROM_TEMPLATE` is set, the terraform
templates will be changed on-the-fly, to comment out the definitions of org and vdc. This will force the test to
borrow org and vcd from the provider.
* `VCD_TEST_SUITE_CLEANUP=1` will clean up testing resources that were created in previous test runs.
* `VCD_TEST_VERBOSE=1` (`-vcd-verbose`) enables verbose output in some tests, such as the list of used tags, or the version
used in the documentation index.
* `VCD_TEST_ORG_USER=1` (`-vcd-test-org-user`) will enable tests with Org User, using the credentials from the configuration file
  (`testEnvBuild.OrgUser` and `testEnvBuild.OrgUserPassword`)
* `VCD_TOKEN=string` : specifies the authentication token to use instead of username/password
   (Use `./scripts/get_token.sh` to retrieve one)
* `VCD_TEST_DISTRIBUTED_NETWORK=1` (`-vcd-test-distributed`) runs testing of distributed networks (requires the edge gateway to have distributed
  routing enabled)
* `VCD_TEST_DATA_GENERATION=1` generates some sample catalog items for data source filter engine test
* `GOVCD_KEEP_TEST_OBJECTS=1` does not delete test objects created with `VCD_TEST_DATA_GENERATION`
* `VCD_MAX_ITEMS=number` during filter engine tests, limits the collection of data sources of a given type to the number
  indicated. The default is 5. The maximum is 100.
* `VCD_PRE_POST_CHECKS` (`-vcd-pre-post-checks`) Perform checks before and after tests (false)
* `VCD_RE_RUN_FAILED` (`-vcd-re-run-failed`) Run only tests that failed in a previous run (false)
* `VCD_REMOVE_TEST_LIST` (`-vcd-remove-test-list`) Remove list of test runs (false)
* `VCD_SHOW_COUNT` (`-vcd-show-count`) Show number of pass/fail tests (false)
* `VCD_SHOW_ELAPSED_TIME` (`-vcd-show-elapsed-time`) Show elapsed time since the start of the suite in pre and post checks (false)
* `VCD_SHOW_TIMESTAMP` (`-vcd-show-timestamp`) Show timestamp in pre and post checks (false)
* `VCD_SKIP_PATTERN` (`-vcd-skip-pattern`) Skip tests that match the pattern (implies vcd-pre-post-checks ()
* `VCD_SKIP_LEFTOVERS_REMOVAL` (`-vcd-skip-leftover-removal`) Do not run the leftovers removal at the end of the suite
* `VCD_SILENT_LEFTOVERS_REMOVAL` (`-vcd-silent-leftover-removal`) Omit details during leftovers removal.
* `VCD_PARTITIONS` (`-vcd-partitions`) Number of partitions used to run the tests
* `VCD_PARTITION_NODE` (`vcd-partition-node`) Number of current node running one of the partitions
* `VCD_PARTITION_TESTS_FILE` (`-vcd-partition-tests-file`) File containing the list of tests that this node will run


When both the environment variable and the command line option are possible, the environment variable gets evaluated first.

## Troubleshooting code issues

### Functions for dumping state and pause during acceptance testing

These functions match signature of Terraform's own `resource.TestCheckResourceAttr` and can be
dropped in for troubleshooting problems. 

This function will dump the state at the test run (while executing all field evaluations). It can
help troubleshooting why some fields fail and find typos, wrong state, etc.

```go
func stateDumper() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		spew.Dump(s)
		return nil
	}
}
```

This function can pause test run in the middle which gives the chance to investigate environment
(UI, API calls, etc)

```go
func sleepTester(d time.Duration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		fmt.Printf("sleeping %s\n", d.String())
		time.Sleep(d)
		fmt.Println("finished sleeping")
		return nil
	}
}
```
