import React from "react";
import { Link } from "react-router-dom";
import gql from "graphql-tag";
import { withStyles } from "@material-ui/core/styles";

import Card from "@material-ui/core/Card";
import CardContent from "@material-ui/core/CardContent";
import Chip from "@material-ui/core/Chip";
import Typography from "@material-ui/core/Typography";

const styles = theme => ({
  labelList: {
    display: "flex",
    flexWrap: "wrap",
    marginTop: theme.spacing.unit
  },
  label: {
    marginRight: theme.spacing.unit
  },
  summary: {
    marginBottom: theme.spacing.unit * 2
  }
});

const BugSummary = ({ bug, classes }) => (
  <Card className={classes.summary}>
    <CardContent>
      <Typography variant="headline" component="h2">
        {bug.title}
      </Typography>
      <Typography variant="subheading" component="h3" title={bug.id}>
        <Link to={"/bug/" + bug.id.slice(0, 8)}>#{bug.id.slice(0, 8)}</Link> •{" "}
        {bug.status.toUpperCase()}
      </Typography>
      <div className={classes.labelList}>
        {bug.labels.map(label => (
          <Chip key={label} label={label} className={classes.label} />
        ))}
      </div>
    </CardContent>
  </Card>
);

BugSummary.fragment = gql`
  fragment BugSummary on Bug {
    id
    title
    status
    labels
  }
`;

export default withStyles(styles)(BugSummary);
