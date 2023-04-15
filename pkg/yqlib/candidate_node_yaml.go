package yqlib

import (
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

func MapYamlStyle(original yaml.Style) Style {
	switch original {
	case yaml.TaggedStyle:
		return TaggedStyle
	case yaml.DoubleQuotedStyle:
		return DoubleQuotedStyle
	case yaml.SingleQuotedStyle:
		return SingleQuotedStyle
	case yaml.LiteralStyle:
		return LiteralStyle
	case yaml.FoldedStyle:
		return FoldedStyle
	case yaml.FlowStyle:
		return FlowStyle
	}
	return 0
}

func MapToYamlStyle(original Style) yaml.Style {
	switch original {
	case TaggedStyle:
		return yaml.TaggedStyle
	case DoubleQuotedStyle:
		return yaml.DoubleQuotedStyle
	case SingleQuotedStyle:
		return yaml.SingleQuotedStyle
	case LiteralStyle:
		return yaml.LiteralStyle
	case FoldedStyle:
		return yaml.FoldedStyle
	case FlowStyle:
		return yaml.FlowStyle
	}
	return 0
}

func (o *CandidateNode) copyFromYamlNode(node *yaml.Node, anchorMap map[string]*CandidateNode) {
	o.Style = MapYamlStyle(node.Style)

	o.Tag = node.Tag
	o.Value = node.Value
	o.Anchor = node.Anchor

	if o.Anchor != "" {
		anchorMap[o.Anchor] = o
		log.Debug("set anchor %v to %v", o.Anchor, NodeToString(o))
	}

	// its a single alias
	if node.Alias != nil && node.Alias.Anchor != "" {
		o.Alias = anchorMap[node.Alias.Anchor]
		log.Debug("set alias to %v", NodeToString(anchorMap[node.Alias.Anchor]))
	}

	o.HeadComment = node.HeadComment
	o.LineComment = node.LineComment
	o.FootComment = node.FootComment

	o.Line = node.Line
	o.Column = node.Column
}

func (o *CandidateNode) copyToYamlNode(node *yaml.Node) {
	node.Style = MapToYamlStyle(o.Style)

	node.Tag = o.Tag
	node.Value = o.Value
	node.Anchor = o.Anchor

	// node.Alias = TODO - find Alias in our own structure
	// might need to be a post process thing

	node.HeadComment = o.HeadComment

	node.LineComment = o.LineComment
	node.FootComment = o.FootComment

	node.Line = o.Line
	node.Column = o.Column
}

func (o *CandidateNode) decodeIntoChild(childNode *yaml.Node, anchorMap map[string]*CandidateNode) (*CandidateNode, error) {
	newChild := o.CreateChild()

	// null yaml.Nodes to not end up calling UnmarshalYAML
	// so we call it explicitly
	if childNode.Tag == "!!null" {
		newChild.Kind = ScalarNode
		newChild.copyFromYamlNode(childNode, anchorMap)
		return newChild, nil
	}

	err := newChild.UnmarshalYAML(childNode, anchorMap)
	return newChild, err
}

func (o *CandidateNode) UnmarshalYAML(node *yaml.Node, anchorMap map[string]*CandidateNode) error {

	log.Debugf("UnmarshalYAML %v", node.Tag)
	switch node.Kind {
	case yaml.DocumentNode:
		log.Debugf("UnmarshalYAML -  a document")
		o.Kind = DocumentNode
		o.copyFromYamlNode(node, anchorMap)
		if len(node.Content) == 0 {
			return nil
		}

		singleChild, err := o.decodeIntoChild(node.Content[0], anchorMap)

		if err != nil {
			return err
		}
		o.Content = []*CandidateNode{singleChild}
		log.Debugf("UnmarshalYAML -  finished document node")
		return nil
	case yaml.AliasNode:
		log.Debug("UnmarshalYAML - alias from yaml: %v", o.Tag)
		o.Kind = AliasNode
		o.copyFromYamlNode(node, anchorMap)
		return nil
	case yaml.ScalarNode:
		log.Debugf("UnmarshalYAML -  a scalar")
		o.Kind = ScalarNode
		o.copyFromYamlNode(node, anchorMap)
		return nil
	case yaml.MappingNode:
		log.Debugf("UnmarshalYAML -  a mapping node")
		o.Kind = MappingNode
		o.copyFromYamlNode(node, anchorMap)
		o.Content = make([]*CandidateNode, len(node.Content))
		for i := 0; i < len(node.Content); i += 2 {

			keyNode, err := o.decodeIntoChild(node.Content[i], anchorMap)
			if err != nil {
				return err
			}

			keyNode.IsMapKey = true

			valueNode, err := o.decodeIntoChild(node.Content[i+1], anchorMap)
			if err != nil {
				return err
			}

			valueNode.Key = keyNode

			o.Content[i] = keyNode
			o.Content[i+1] = valueNode
		}
		log.Debugf("UnmarshalYAML -  finished mapping node")
		return nil
	case yaml.SequenceNode:
		log.Debugf("UnmarshalYAML -  a sequence: %v", len(node.Content))
		o.Kind = SequenceNode
		o.copyFromYamlNode(node, anchorMap)
		o.Content = make([]*CandidateNode, len(node.Content))
		for i := 0; i < len(node.Content); i += 1 {
			keyNode := o.CreateChild()
			keyNode.IsMapKey = true // can't remember if we need this for sequences
			keyNode.Tag = "!!int"
			keyNode.Kind = ScalarNode
			keyNode.Value = fmt.Sprintf("%v", i)

			valueNode, err := o.decodeIntoChild(node.Content[i], anchorMap)
			if err != nil {
				return err
			}

			valueNode.Key = keyNode
			o.Content[i] = valueNode
		}
		return nil
	case 0:
		// not sure when this happens
		o.copyFromYamlNode(node, anchorMap)
		log.Debugf("UnmarshalYAML -  errr.. %v", NodeToString(o))
		return nil
	default:
		return fmt.Errorf("orderedMap: invalid yaml node")
	}
}

func (o *CandidateNode) MarshalYAML() (*yaml.Node, error) {
	log.Debug("MarshalYAML to yaml: %v", o.Tag)
	switch o.Kind {
	case DocumentNode:
		log.Debug("MarshalYAML its a document")
		target := &yaml.Node{Kind: yaml.DocumentNode}
		o.copyToYamlNode(target)

		singleChild, err := o.Content[0].MarshalYAML()

		log.Debug("MarshalYAML its a document - singChild is %v", singleChild)
		if err != nil {
			return nil, err
		}
		target.Content = []*yaml.Node{singleChild}
		return target, nil
	case AliasNode:
		log.Debug("MarshalYAML - alias to yaml: %v", o.Tag)
		target := &yaml.Node{Kind: yaml.AliasNode}
		o.copyToYamlNode(target)
		return target, nil
	case ScalarNode:
		log.Debug("MarshalYAML - scalar: %v", o.Value)
		target := &yaml.Node{Kind: yaml.ScalarNode}
		o.copyToYamlNode(target)
		return target, nil
	case MappingNode, SequenceNode:
		targetKind := yaml.MappingNode
		if o.Kind == SequenceNode {
			targetKind = yaml.SequenceNode
		}
		target := &yaml.Node{Kind: targetKind}
		o.copyToYamlNode(target)
		target.Content = make([]*yaml.Node, len(o.Content))
		for i := 0; i < len(o.Content); i += 1 {

			child, err := o.Content[i].MarshalYAML()

			if err != nil {
				return nil, err
			}
			target.Content[i] = child
		}
		return target, nil
	}
	target := &yaml.Node{}
	o.copyToYamlNode(target)
	return target, nil
}