import { ArrayField, ChipField, Datagrid, List, ReferenceField, SingleFieldList, TextField, useGetList } from "react-admin";
import Tree from 'react-d3-tree';

const orgChart = {
    name: 'CEO',
    children: [
        {
            name: 'Manager',
            attributes: {
                department: 'Production',
            },
            children: [
                {
                    name: 'Foreman',
                    attributes: {
                        department: 'Fabrication',
                    },
                    children: [
                        {
                            name: 'Worker',
                        },
                    ],
                },
                {
                    name: 'Foreman',
                    attributes: {
                        department: 'Assembly',
                    },
                    children: [
                        {
                            name: 'Worker',
                        },
                    ],
                },
            ],
        },
    ],
};

// rootNodes returns all the root nodes in a graph
const rootNodes = (graph: Array<any>) => {
    return graph.filter(
        (node) => node.inputs.map((i) => i.root).includes(true)
    );
}

// buildTree takes a root node and a graph and returns a tree
const buildTree = (root: any, graph: Array<any>) => {
    const children = graph.filter(
        (node) => node.inputs.map((i) => i.step_id).includes(root.name)
    );
    if (children.length === 0) {
        return root;
    }
    root.children = children.map((d) => buildTree({
        name: d.id, attributes: {
            job_id: d.job_id,
        }
    }, graph));
    return root;
}

const DAG = () => {
    const { data, total, isLoading, error, refetch } = useGetList("graph");
    if (!data) return null;
    const roots = rootNodes(data);
    const chart = roots.map((d) => buildTree({ name: d.id }, data));
    var graphList = chart.map((d) =>
        <Tree key={d} data={d} />
    );
    return (
        <div>
            {graphList}
        </div>
    );
};

export const GraphList = () => (
    <div>
        <DAG />
        <List>
            <Datagrid rowClick="show">
                <TextField source="id" />
                <ArrayField source="inputs">
                    <SingleFieldList><ChipField source="step_id" /></SingleFieldList>
                </ArrayField>
                <TextField source="job_id" />
            </Datagrid>
        </List>
    </div>
);
