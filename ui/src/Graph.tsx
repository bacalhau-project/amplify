import { ArrayField, ChipField, Datagrid, List, SingleFieldList, TextField } from "react-admin";
import { DAG } from './CenteredTree';

export const GraphList = () => (
    <div>
        <DAG />
        <List>
            <Datagrid rowClick="show">
                <TextField source="id" />
                <ArrayField source="attributes.inputs">
                    <SingleFieldList><ChipField source="node_id" /></SingleFieldList>
                </ArrayField>
                <TextField source="attributes.job_id" />
            </Datagrid>
        </List>
    </div>
);
