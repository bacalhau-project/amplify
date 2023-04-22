import { select } from 'd3-selection';
import { useLayoutEffect, useRef, useState } from 'react';
import { useGetList } from "react-admin";
import Tree from "react-d3-tree";

// rootNodes returns all the root nodes in a graph
const rootNodes = (graph: Array<any>) => {
    return graph.filter(
        (node) => node.attributes.inputs.map((i) => i.root).includes(true)
    );
}

// buildTree takes a root node and a graph and returns a tree
const buildTree = (root: any, graph: Array<any>) => {
    const children = graph.filter(
        (node) => node.attributes.inputs.map((i) => i.node_id).includes(root.name)
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

const containerStyles = {
    width: '100%',
    height: '50vh',
}

export const DAG = () => {
    const { data, total, isLoading, error, refetch } = useGetList("graph");
    if (!data) return null;
    const roots = rootNodes(data);
    const chart = roots.map((d) => buildTree({ name: d.id }, data));
    var graphList = chart.map((d) =>
        <CenteredTree key={d} data={d} />
    );
    return (
        <div >
            {graphList}
        </div>
    );
};

const CenteredTree = props => {
    const ref = useRef<any>(null);
    const treeRef = useRef<any>(null);
    const [translate, setTranslation] = useState({ x: 0, y: 0 });
    const [zoom, setZoom] = useState(1);

    useLayoutEffect(() => {
        if (!ref.current || !treeRef.current) return;
        const dimensions = ref.current.getBoundingClientRect();
        const svgRef = select("." + treeRef.current.svgInstanceRef);
        const svgDimensions = svgRef.node().getBBox();
        var zoom = dimensions.height / svgDimensions.height;
        zoom = zoom - zoom * (dimensions.height / 10) / dimensions.height;
        setZoom(zoom);
        setTranslation({
            x: dimensions.width / 2,
            y: dimensions.height / 10
        });
    }, []);

    return (
        <div style={containerStyles} ref={ref}>
            <Tree
                ref={treeRef}
                data={props.data}
                translate={translate}
                orientation={'vertical'}
                zoom={zoom}
            />
        </div>
    );
}
