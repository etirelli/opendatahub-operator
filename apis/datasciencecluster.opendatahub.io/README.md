# DataScienceCluster CRD

This document explains all the fields defined by the DataScienceCluster CRD. This crd is used for deployment of
ODH components like notebooks, serving and training.

## Goals

Following is the list of goals for this crd

- Enable / Disable individual components provided by ODH 
- Run components with optional dashboard integration
- Run components with optional monitoring integration
- Allow cluster admins to set controller resources for individual components
- Allow cluster admins to set controller replicas for individual components
- Allow cluster admins to set component specific configurations for individual components
- Allow cluster admins to set Oauth options for all components


## spec.components

This is a list of different component profiles provided by ODH. Note these component profiles can be group of multiple
controllers and custom resources. Every component has the following common fields

- `enabled` : When set to true, all the component resources are deployed.
- `replicas` : When set to an int value, component controllers will scale to the given value
- `resources`: Cluster admin can set this value as defined [here](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#resources).
- `dashboard`: When set to true, component integration with dashboard will be enabled.
- `monitoring`: When set to true, component integration with monitoring resources(prometheus, grafana) will be enabled.

### spec.components.notebooks

In addition to above fields, `notebooks` has the following component specific fields -
- `notebookImages.managed` : When set to true, will allow users to use notebooks provided by ODH

### spec.components.serving

In addition to above fields, `serving` has the following component specific fields -
- TBD

### spec.components.training

In addition to above fields, `training` has the following component specific fields -
- TBD