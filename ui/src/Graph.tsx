import { ArrayField, ChipField, Datagrid, List, SingleFieldList, TextField } from "react-admin";
import { DAG } from './CenteredTree';

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
