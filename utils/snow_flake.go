package utils

import sf "github.com/bwmarrin/snowflake"

var node *sf.Node

func GetNode() *sf.Node {
	return node
}

func InitSnowflake(nodeID int64) error {
	var err error
	node, err = sf.NewNode(nodeID)
	return err
}

func GenerateSnowflakeID() int64 {
	if node == nil {
		panic("snowflake not initialized")
	}
	return node.Generate().Int64()
}
